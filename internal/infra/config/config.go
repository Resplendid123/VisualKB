package config

import (
	"time"
)

type Config struct {
	HTTP         HTTPConfig         `yaml:"http"`
	Database     DatabaseConfig     `yaml:"database"`
	Redis        RedisConfig        `yaml:"redis"`
	Auth         AuthConfig         `yaml:"auth"`
	S3           S3Config           `yaml:"s3"`
	LLM          LLMConfig          `yaml:"llm"`
	Embedding    EmbeddingConfig    `yaml:"embedding"`
	Rerank       RerankConfig       `yaml:"rerank"`
	Query        QueryConfig        `yaml:"query"`
	Observe      ObserveConfig      `yaml:"observe"`
	Logging      LoggingConfig      `yaml:"logging"`
	Sandbox      SandboxConfig      `yaml:"sandbox"`
	Skills       SkillsConfig       `yaml:"skills"`
	IngestWorker IngestWorkerConfig `yaml:"ingest_worker"`
	Queue        QueueConfig        `yaml:"queue"`
	SessionLock  SessionLockConfig  `yaml:"session_lock"`
}

type HTTPConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	DB          string `yaml:"database"`
	SSLMode     string `yaml:"sslmode"`
	MaxOpenConn int    `yaml:"max_open_conns"`
	MaxIdleConn int    `yaml:"max_idle_conns"`
}
type RedisConfig struct {
	Addr        string        `yaml:"addr"`
	Password    string        `yaml:"password"`
	DB          int           `yaml:"database"`
	DialTimeout time.Duration `yaml:"dial_timeout"`
}

type AuthConfig struct {
	JWTSecret     string        `yaml:"jwt_secret"`
	Issuer        string        `yaml:"issuer"`
	TTL           time.Duration `yaml:"ttl"`
	RefreshWindow time.Duration `yaml:"refresh_window"`
	CookieSecure  bool          `yaml:"cookie_secure"`
}

type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}
type LLMConfig struct {
	Protocol      string `yaml:"protocol"`
	BaseURL       string `yaml:"base_url"`
	ApiKey        string `yaml:"api_key"`
	Model         string `yaml:"model"`
	ContextWindow int    `yaml:"context_window"`
	MaxTokens     int    `yaml:"max_tokens"`
}
type EmbeddingConfig struct {
	BaseURL   string `yaml:"base_url"`
	ApiKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	Dimension int    `yaml:"dimension"`
}
type RerankConfig struct {
	BaseURL string `yaml:"base_url"`
	ApiKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	TopN    int    `yaml:"top_n"`
}

type QueryConfig struct {
	Strategy string `yaml:"strategy"`
	MultiN   int    `yaml:"multi_n"`
}
type ObserveConfig struct {
	Endpoint       string `yaml:"endpoint"`
	ServiceName    string `yaml:"service_name"`
	ServiceVersion string `yaml:"service_version"`
	Enabled        bool   `yaml:"enabled"`
}
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type SandboxConfig struct {
	// Drop whole ns after idle TTL.
	NamespaceIdleTTL time.Duration `yaml:"namespace_idle_ttl"`
	PodReadyTimeout  time.Duration `yaml:"pod_ready_timeout"`
}

type SkillsConfig struct {
	Dir string `yaml:"dir"`
}

type IngestWorkerConfig struct {
	Interval      time.Duration `yaml:"interval"`
	BatchLimit    int           `yaml:"batch_limit"`
	PerDocTimeout time.Duration `yaml:"per_doc_timeout"`
}

type QueueConfig struct {
	Concurrency     int           `yaml:"concurrency"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// Lock TTL = agent ctx deadline.
type SessionLockConfig struct {
	TTL time.Duration `yaml:"ttl"`
}
