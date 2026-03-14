package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort int
	GraderAddr string
	Judge      *JudgeConfig
	Minio      *MinioConfig
	GCS        *GCSConfig
	PubSub     *PubSubConfig
	RabbitMQ   *RabbitMQConfig
}

type JudgeConfig struct {
	SubmissionsDir  string
	LibcontainerDir string
	ImagesDir       string
	OverlayFSDir    string
	RootfsDir       string
	WorkRoot        string
	MaxConcurrency  int
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type GCSConfig struct {
	Bucket          string
	ProjectID       string
	CredentialsFile string
}

type PubSubConfig struct {
	ProjectID          string
	CredentialsFile    string
	SubscriptionSuffix string
}

type RabbitMQConfig struct {
	URL             string
	Queue           string
	QueueDurable    bool
	QueueAutoDelete bool
	PrefetchCount   int
}

func LoadConfig() *Config {
	if os.Getenv("ENV") == "dev" {
		godotenv.Load()
	}

	return &Config{
		ServerPort: getEnvInt("SERVER_PORT", 8080),
		GraderAddr: getEnv("GRADER_ADDR", "localhost:8080"),
		Judge: &JudgeConfig{
			SubmissionsDir:  getEnv("JUDGE_SUBMISSIONS_DIR", "/tmp/judge/submissions"),
			LibcontainerDir: getEnv("JUDGE_LIBCONTAINER_DIR", "/tmp/judge/libcontainer"),
			ImagesDir:       getEnv("JUDGE_IMAGES_DIR", "/var/lib/judge/images"),
			OverlayFSDir:    getEnv("JUDGE_OVERLAYFS_DIR", "/tmp/judge/overlayfs"),
			RootfsDir:       getEnv("JUDGE_ROOTFS_DIR", "/tmp/judge/rootfs"),
			WorkRoot:        getEnv("JUDGE_WORK_ROOT", "/tmp/judge/work"),
			MaxConcurrency:  getEnvInt("JUDGE_MAX_CONCURRENCY", 4),
		},
		Minio: &MinioConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "jjudge"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		GCS: &GCSConfig{
			Bucket:          getEnv("GCS_BUCKET", ""),
			ProjectID:       getEnv("GCS_PROJECT_ID", ""),
			CredentialsFile: getEnv("GCS_CREDENTIALS_FILE", ""),
		},
		PubSub: &PubSubConfig{
			ProjectID:          getEnv("PUBSUB_PROJECT_ID", ""),
			CredentialsFile:    getEnv("PUBSUB_CREDENTIALS_FILE", ""),
			SubscriptionSuffix: getEnv("PUBSUB_SUBSCRIPTION_SUFFIX", "-sub"),
		},
		RabbitMQ: &RabbitMQConfig{
			URL:             getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Queue:           getEnv("RABBITMQ_QUEUE", "submissions"),
			QueueDurable:    getEnv("RABBITMQ_QUEUE_DURABLE", "false") == "true",
			QueueAutoDelete: getEnv("RABBITMQ_QUEUE_AUTO_DELETE", "false") == "true",
			PrefetchCount:   getEnvInt("RABBITMQ_PREFETCH_COUNT", 0),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		var value int
		fmt.Sscanf(valueStr, "%d", &value)
		return value
	}
	return defaultValue
}
