package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort           string
	PostgresDSN        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string
	IdempotencyTTL     time.Duration
	IdempotencyLockTTL time.Duration
}

func Load() Config {
	return Config{
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/txdb?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		KafkaBrokers:       []string{getEnv("KAFKA_BROKER", "localhost:9092")},
		KafkaTopic:         getEnv("KAFKA_TOPIC", "transaction-events"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "transaction-consumer"),
		IdempotencyTTL:     getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),
		IdempotencyLockTTL: getEnvDuration("IDEMPOTENCY_LOCK_TTL", 30*time.Second),
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}
