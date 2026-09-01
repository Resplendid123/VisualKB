package main

import (
	"context"
	"fmt"
	"os"
	"time"

	authapp "learn/internal/application/auth"
	convapp "learn/internal/application/conversation"
	documentapp "learn/internal/application/document"
	projectapp "learn/internal/application/project"
	userapp "learn/internal/application/user"
	httpsrv "learn/internal/interfaces/http"
	authhandler "learn/internal/interfaces/http/auth"

	"learn/internal/infra/ai"
	"learn/internal/infra/ai/chunk"
	"learn/internal/infra/ai/embed"
	_ "learn/internal/infra/ai/extract"
	"learn/internal/infra/ai/ingest"
	"learn/internal/infra/ai/llm"
	"learn/internal/infra/ai/query"
	"learn/internal/infra/ai/rerank"
	"learn/internal/infra/ai/retrieve"
	"learn/internal/infra/ai/skills"
	"learn/internal/infra/ai/tools"
	authinfra "learn/internal/infra/auth"
	"learn/internal/infra/cache"
	"learn/internal/infra/config"
	"learn/internal/infra/data"
	"learn/internal/infra/data/repo"
	"learn/internal/infra/database"
	"learn/internal/infra/queue"
	"learn/internal/infra/queue/handlers"
	redisinfra "learn/internal/infra/redis"
	"learn/internal/infra/s3"
	"learn/internal/infra/sandbox"

	"log/slog"

	"github.com/hibiken/asynq"
)

type Application struct {
	HTTP    *httpsrv.Server
	Cleanup func()
}

func Bootstrap(ctx context.Context, cfg *config.Config) (*Application, error) {
	db, closeDB, err := database.OpenDB(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := data.AutoMigrate(ctx, db); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	rdb, closeRedis, err := cache.OpenRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("open redis: %w", err)
	}

	// K8s applies deferred to sandbox controller.
	ctrlExec, err := sandbox.NewControllerExecutor()
	if err != nil {
		return nil, fmt.Errorf("init sandbox controller executor: %w", err)
	}

	if err := skills.Default.Load(cfg.Skills.Dir); err != nil {
		return nil, fmt.Errorf("load skills from %s: %w", cfg.Skills.Dir, err)
	}
	authRepo := authinfra.NewJWTAdapter(cfg.Auth)
	userRepo := repo.NewUserRepo(db)
	convoRepo := repo.NewConvoRepo(db)
	msgRepo := repo.NewMsgRepo(db)
	msgSeqRepo := repo.NewMsgSeqRepo(rdb, db)
	streamRepo := repo.NewEventStreamRepo(rdb)
	msgCacheRepo := repo.NewMsgCacheRepo(rdb)
	sandboxRepo := repo.NewSandboxRepo(db)
	execRepo := repo.NewExecutionRepo(db)
	projectRepo := repo.NewProjectRepo(db)
	docRepo := repo.NewDocumentRepo(db)
	docVerRepo := repo.NewDocumentVersionRepo(db)
	docChunkRepo := repo.NewDocumentChunkRepo(db)
	treeRepo := repo.NewTreeRepo(db)

	// Distributed lock plus cross-instance cancel pub/sub.
	sessionLock := redisinfra.NewSessionLock(rdb, cfg.SessionLock.TTL)
	cancelBus := redisinfra.NewCancelBus(rdb, instanceID())

	llmClient := llm.NewOpenAI(cfg.LLM)
	embedder := embed.New(cfg.Embedding, 60*time.Second)
	rerankLLMClient := llm.NewOpenAI(config.LLMConfig{
		BaseURL: cfg.Rerank.BaseURL,
		ApiKey:  cfg.Rerank.ApiKey,
		Model:   cfg.Rerank.Model,
	})
	reranker := rerank.New(rerankLLMClient, cfg.Rerank.Model)
	transformer := query.NewTransformer(llmClient, query.Strategy(cfg.Query.Strategy), cfg.Query.MultiN)
	searcher := retrieve.NewSearcherFull(db, embedder, reranker, transformer)
	objectStore, err := s3.New(cfg.S3)
	if err != nil {
		return nil, fmt.Errorf("init s3: %w", err)
	}
	docSplitter := chunk.New()
	docIngestor := ingest.New(docRepo, docVerRepo, docChunkRepo, objectStore, docSplitter, embedder)

	userSvc := userapp.NewUserService(userRepo)
	authSvc := authapp.NewAuthService(userRepo, authRepo)

	sandboxSvc := projectapp.NewSandboxService(
		sandboxRepo, execRepo,
		ctrlExec, ctrlExec,
		cfg.Sandbox,
	)
	// Sandbox lifecycle bound to project, not conversation.
	projectSvc := projectapp.NewProjectService(projectRepo, convoRepo, sandboxSvc)

	tools.RegisterBashTool(sandboxSvc)
	tools.RegisterProjectTools(projectSvc)
	tools.RegisterRetrieveTool(searcher)
	tools.RegisterSkillLoaderTool(skills.Default)
	tools.RegisterAskUserTool()
	tools.RegisterWriteMemoryTool(userSvc)
	agent := convapp.NewAgent(llmClient, tools.Default)
	agent.SetConvoRepo(convoRepo)

	portraitLoader := func(ctx context.Context, uid int64) string {
		immutable, mutable, err := userSvc.GetPortrait(ctx, uid)
		if err != nil {
			slog.WarnContext(ctx, "load user portrait failed; continuing with empty", "user_id", uid, "err", err)
			return ""
		}
		return fmt.Sprintf("[Immutable — user-edited]\n%s\n\n[Mutable — agent-written]\n%s", immutable, mutable)
	}

	// Conversation layer holds no sandbox dependency.
	conversationSvc := convapp.NewConversationService(
		convoRepo, msgRepo, msgSeqRepo, streamRepo, msgCacheRepo,
		sessionLock, cancelBus,
		agent, skills.Default, ai.RenderSystemPrompt,
		docRepo, portraitLoader,
	)
	// Listener ctx follows process lifecycle.
	cancelCleanup := conversationSvc.StartCancelListener(ctx)
	defer func() {
		if err := cancelCleanup(); err != nil {
			slog.Error("cancel listener cleanup failed", "err", err)
		}
	}()

	documentSvc := documentapp.NewDocumentService(docRepo, docVerRepo, treeRepo, objectStore, docIngestor)
	tools.RegisterDocumentTools(documentSvc, documentSvc, documentSvc)
	treeSvc := documentapp.NewTreeService(treeRepo, documentSvc)

	q, err := queue.New(queue.Config{
		RedisAddr:       cfg.Redis.Addr,
		RedisPassword:   cfg.Redis.Password,
		RedisDB:         cfg.Redis.DB,
		Concurrency:     cfg.Queue.Concurrency,
		ShutdownTimeout: cfg.Queue.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("init queue: %w", err)
	}

	q.Mux().Handle(handlers.TypeIngestSweep, handlers.NewIngestSweepHandler(
		documentSvc, cfg.IngestWorker.BatchLimit, cfg.IngestWorker.PerDocTimeout,
	))
	// Idle pod reclaim owned by controller.

	if cfg.IngestWorker.Interval > 0 {
		if _, err := q.Scheduler().Register("@every "+cfg.IngestWorker.Interval.String(),
			asynq.NewTask(handlers.TypeIngestSweep, nil, asynq.MaxRetry(3))); err != nil {
			return nil, fmt.Errorf("register ingest sweep: %w", err)
		}
	}

	if err := q.Start(ctx); err != nil {
		return nil, fmt.Errorf("start queue: %w", err)
	}

	cookie := authhandler.AuthCookie{
		Name:   "access_token",
		TTL:    cfg.Auth.TTL,
		Secure: cfg.Auth.CookieSecure,
	}
	httpCfg := httpsrv.HTTPConfig{
		Addr:         cfg.HTTP.Addr,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		ServiceName:  cfg.Observe.ServiceName,
	}
	handlers := httpsrv.NewHandlers(userSvc, conversationSvc, authSvc, projectSvc, documentSvc, treeSvc, cookie)

	srv := httpsrv.NewServer(httpCfg, handlers, authSvc)
	return &Application{
		HTTP: srv,
		Cleanup: func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				slog.Error("server shutdown failed", "err", err)
			}
			q.Shutdown()
			if err := closeRedis(); err != nil {
				slog.Error("close redis failed", "err", err)
			}
			if err := closeDB(); err != nil {
				slog.Error("close database failed", "err", err)
			}
		},
	}, nil
}

// instanceID identifies this instance; defaults "local".
func instanceID() string {
	if v := os.Getenv("INSTANCE_ID"); v != "" {
		return v
	}
	if v := os.Getenv("POD_NAME"); v != "" {
		return v
	}
	if v := os.Getenv("HOSTNAME"); v != "" {
		return v
	}
	return "local"
}
