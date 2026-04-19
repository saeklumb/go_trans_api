package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go-project/internal/config"
	"go-project/internal/kafka"
	"go-project/internal/migrations"
	"go-project/internal/repository/postgres"
	redisrepo "go-project/internal/repository/redis"
	"go-project/internal/service"
	httptransport "go-project/internal/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := migrations.Run(cfg.PostgresDSN); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("postgres connect error: %v", err)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	if err = rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer producer.Close()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaConsumerGroup)
	defer consumer.Close()
	go consumer.Run(ctx)

	store := postgres.NewStore(pool)
	idempotencyStore := redisrepo.NewIdempotencyStore(rdb)
	svc := service.NewTransferService(store, idempotencyStore, producer, cfg.IdempotencyTTL, cfg.IdempotencyLockTTL)
	handler := httptransport.NewHandler(svc)

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: handler.Router(),
	}

	go func() {
		log.Printf("API started on :%s", cfg.HTTPPort)
		if serveErr := srv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatalf("http serve error: %v", serveErr)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
