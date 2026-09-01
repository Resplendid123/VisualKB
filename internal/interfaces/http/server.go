package http

import (
	"context"
	"learn/internal/interfaces/http/middleware"

	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Server struct {
	engine *gin.Engine
	http   *http.Server
	cfg    HTTPConfig
}

// NewServer builds Gin server with middleware.
func NewServer(cfg HTTPConfig, handlers *Handlers, verifier middleware.TokenVerifier) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.Use(otelgin.Middleware(cfg.ServiceName))
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())
	engine.Use(middleware.CORS())

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	api := engine.Group("/api/v1")
	RegisterRoutes(api, handlers, verifier)

	return &Server{
		engine: engine,
		cfg:    cfg,
		http: &http.Server{
			Addr:         cfg.Addr,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			Handler:      engine,
		},
	}
}

func (s *Server) Start() error {
	slog.Info("HTTP server starting", "addr", s.cfg.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

type HTTPConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	ServiceName  string
}
