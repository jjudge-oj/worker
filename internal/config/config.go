package config

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

type Config struct {
	Env               string
	ResultsURL        string
	WorkRoot          string
	RootfsDir         string
	ServerPort        int
	MetricsAddr       string
	QueuePollInterval time.Duration
	Judge             JudgeConfig
	Database          DatabaseConfig
	RabbitMQ          RabbitMQConfig
	ProblemCacheDir   string
	DatabaseURL       string
	Minio             MinioConfig
}

type ServerConfig struct {
}

type JudgeConfig struct {
	DiskCacheDir    string
	SubmissionsDir  string
	LibcontainerDir string
	ImagesDir       string
	OverlayFSDir    string
	RootfsDir       string
	WorkRoot        string
	MaxConcurrency  int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=%s",
		d.User,
		d.Password,
		d.Name,
		d.Host,
		d.Port,
		d.SSLMode,
	)
}

type RabbitMQConfig struct {
	URL   string
	Queue string
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func Load() *Config {
	return &Config{
		Env:               get("ENV", "dev"),
		ResultsURL:        get("RESULTS_URL", "http://backend/results"),
		ServerPort:        getInt("SERVER_PORT", 8000),
		MetricsAddr:       get("METRICS_ADDR", ":9090"),
		QueuePollInterval: time.Duration(getInt64("QUEUE_POLL_MS", 200)) * time.Millisecond,
		RabbitMQ: RabbitMQConfig{
			URL:   get("RABBITMQ_URL", "amqp://castletown:castletown@localhost:5672/"),
			Queue: get("RABBITMQ_QUEUE", "submissions"),
		},
		Database: DatabaseConfig{
			Host:     get("DATABASE_HOST", "localhost"),
			Port:     getInt("DATABASE_PORT", 5432),
			User:     get("DATABASE_USER", "castletown"),
			Password: get("DATABASE_PASSWORD", "castletown"),
			Name:     get("DATABASE_NAME", "castletown"),
			SSLMode:  get("DATABASE_SSLMODE", "disable"),
		},
		Minio: MinioConfig{
			Endpoint:  get("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: get("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: get("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    get("MINIO_BUCKET", "castletown"),
			UseSSL:    get("MINIO_USE_SSL", "false") == "true",
		},
		Judge: JudgeConfig{
			DiskCacheDir:    get("JUDGE_DISK_CACHE_DIR", "/var/castletown/testcases"),
			MaxConcurrency:  getInt("JUDGE_MAX_CONCURRENCY", runtime.NumCPU()),
			SubmissionsDir:  get("JUDGE_SUBMISSIONS_DIR", "/var/castletown/submissions"),
			LibcontainerDir: get("JUDGE_LIBCONTAINER_DIR", "/var/castletown/libcontainer"),
			ImagesDir:       get("JUDGE_IMAGES_DIR", "/var/castletown/images"),
			RootfsDir:       get("JUDGE_ROOTFS_DIR", "/tmp/castletown/rootfs"),
			WorkRoot:        get("JUDGE_WORK_ROOT", "/tmp/castletown/work"),
			OverlayFSDir:    get("JUDGE_OVERLAYFS_DIR", "/tmp/castletown/overlayfs"),
		},
	}
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	return int(getInt64(k, int64(def)))
}

func getInt64(k string, def int64) int64 {
	if v := os.Getenv(k); v != "" {
		if x, err := parseInt64(v); err == nil {
			return x
		}
	}
	return def
}

func parseInt64(s string) (int64, error) { var x int64; _, err := fmt.Sscan(s, &x); return x, err }
