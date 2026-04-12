package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	HTTPAddr string

	Postgres        PostgresConfig
	PostgresReplica PostgresReplicaConfig
	Projector       ProjectorConfig
	Etcd            EtcdConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
	SSLMode  string
}

type PostgresReplicaConfig struct {
	Enabled bool
	PostgresConfig
}

type ProjectorConfig struct {
	BatchSize    int
	PollInterval time.Duration
}

type EtcdConfig struct {
	Enabled     bool
	Endpoints   []string
	DialTimeout time.Duration
	KeyPrefix   string
}

func Load() Config {
	return Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			DB:       getEnv("POSTGRES_DB", "learning_marketplace"),
			User:     getEnv("POSTGRES_USER", "app"),
			Password: getEnv("POSTGRES_PASSWORD", "app"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		PostgresReplica: PostgresReplicaConfig{
			Enabled: getEnvBool("POSTGRES_REPLICA_ENABLED", false),
			PostgresConfig: PostgresConfig{
				Host:     getEnv("POSTGRES_REPLICA_HOST", "localhost"),
				Port:     getEnv("POSTGRES_REPLICA_PORT", "5432"),
				DB:       getEnv("POSTGRES_REPLICA_DB", "learning_marketplace"),
				User:     getEnv("POSTGRES_REPLICA_USER", "app"),
				Password: getEnv("POSTGRES_REPLICA_PASSWORD", "app"),
				SSLMode:  getEnv("POSTGRES_REPLICA_SSLMODE", "disable"),
			},
		},
		Projector: ProjectorConfig{
			BatchSize:    getEnvInt("PROJECTOR_BATCH_SIZE", 50),
			PollInterval: getEnvDuration("PROJECTOR_POLL_INTERVAL", 2*time.Second),
		},
		Etcd: EtcdConfig{
			Enabled:     getEnvBool("ETCD_ENABLED", false),
			Endpoints:   getEnvList("ETCD_ENDPOINTS", []string{"http://localhost:2379"}),
			DialTimeout: getEnvDuration("ETCD_DIAL_TIMEOUT", 5*time.Second),
			KeyPrefix:   getEnv("ETCD_KEY_PREFIX", "/learning-marketplace/leases/"),
		},
	}
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		p.Host,
		p.Port,
		p.DB,
		p.User,
		p.Password,
		p.SSLMode,
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvList(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}

	return out
}
