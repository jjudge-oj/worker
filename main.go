package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jjudge-oj/worker/config"
	"github.com/jjudge-oj/worker/internal/blob"
	"github.com/jjudge-oj/worker/internal/grader"
	"github.com/jjudge-oj/worker/internal/lime"
	"github.com/jjudge-oj/worker/internal/mq"
	"github.com/jjudge-oj/worker/internal/tccache"
	"github.com/jjudge-oj/worker/internal/worker"
)

func main() {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init blob storage
	blobStorage, err := blob.NewStorageFromConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to init blob storage: %v", err)
	}
	if err := blobStorage.EnsureBucket(ctx); err != nil {
		log.Fatalf("failed to ensure bucket: %v", err)
	}

	// Init testcase cache
	tc, err := tccache.NewTestcaseCache(1000, cfg.Judge.WorkRoot)
	if err != nil {
		log.Fatalf("failed to init testcase cache: %v", err)
	}
	tc.SetBlobStorage(blobStorage)

	// Init MQ client
	mqClient, err := mq.NewRabbitMQClient(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("failed to init rabbitmq: %v", err)
	}
	mqWrapper := mq.New(mqClient)
	defer mqWrapper.Close()

	// Init grader gRPC client
	graderClient, err := grader.NewClient(cfg.GraderAddr)
	if err != nil {
		log.Fatalf("failed to init grader client: %v", err)
	}
	defer graderClient.Close()

	// Init slot pool
	slotPool := lime.NewSlotPool(lime.WithMaxConcurrency(cfg.Judge.MaxConcurrency))

	// Create and start worker
	w := worker.New(cfg, mqWrapper, graderClient, blobStorage, tc, slotPool)

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		cancel()
	}()

	log.Printf("starting worker (max_concurrency=%d, grader=%s)", cfg.Judge.MaxConcurrency, cfg.GraderAddr)
	if err := w.Start(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("worker exited with error: %v", err)
	}
	log.Println("worker stopped")
}
