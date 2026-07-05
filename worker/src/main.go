package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	jobsProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "worker_jobs_processed_total",
			Help: "Total jobs processed",
		},
	)
)

var (
	jobsFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "worker_jobs_failed_total",
			Help: "Total jobs failed",
		},
	)
)

type JobHandler func(ctx context.Context, jobInfo JobInfo) error

func main() {
	log.Println("Worker starting...")

	// Create a new worker
	w, err := NewWorker()
	if err != nil {
		log.Fatalf("Failed to initialize worker: %v", err)
	}

	// Root cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Go(func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	})

	switch os.Getenv("role") {
	case "poller": // Start poller
		wg.Go(func() {
			w.pollPendingJobs(ctx)
		})
	case "worker": // Start DB jobs consumer
		prometheus.MustRegister(jobsProcessed)
		prometheus.MustRegister(jobsFailed)
		w.SetMetrics(jobsProcessed, jobsFailed)
		var handlerMap = map[string]JobHandler{
			"CHANNEL_SEARCH": func(ctx context.Context, jobInfo JobInfo) error {
				return w.channelSearch(ctx, jobInfo.ID, jobInfo.TargetID)
			},
			"KEYWORD_SEARCH": func(ctx context.Context, jobInfo JobInfo) error {
				return w.keywordSearch(ctx, jobInfo.ID, jobInfo.TargetID, jobInfo.Query.String)
			},
			"SEMANTIC_SEARCH": func(ctx context.Context, jobInfo JobInfo) error {
				return w.semanticSearch(ctx, jobInfo.ID, jobInfo.TargetID, jobInfo.Query.String)
			},
			"TOPIC_DETECTION_EMBED": func(ctx context.Context, jobInfo JobInfo) error {
				return w.topicDetectionEmbed(ctx, jobInfo.ID, jobInfo.TargetID)
			},
			"VIDEO_SUMMARIZATION_TRANSCRIBE": func(ctx context.Context, jobInfo JobInfo) error {
				return w.videoSummarizationTranscribe(ctx, jobInfo.ID, jobInfo.TargetID)
			},
		}

		concurrency, err := strconv.Atoi(os.Getenv("worker_concurrency"))
		if err != nil {
			concurrency = 1
		}

		for i := 0; i < concurrency; i++ {
			wg.Go(func() {
				w.executeDBJobs(ctx, "LISTEN jobs_available", keysFromMap(handlerMap), handlerMap)
			})
		}
	case "llm_worker": // Start llm jobs consumer
		prometheus.MustRegister(jobsProcessed)
		prometheus.MustRegister(jobsFailed)
		w.SetMetrics(jobsProcessed, jobsFailed)

		var handlerMap = map[string]JobHandler{
			"TOPIC_DETECTION_LLM": func(ctx context.Context, jobInfo JobInfo) error {
				return w.topicDetectionLLM(ctx, jobInfo.ID)
			},
			"VIDEO_SUMMARIZATION_LLM": func(ctx context.Context, jobInfo JobInfo) error {
				return w.videoSummarizationLLM(ctx, jobInfo.ID)
			},
		}

		concurrency, err := strconv.Atoi(os.Getenv("llm_concurrency"))
		if err != nil {
			concurrency = 1
		}

		for i := 0; i < concurrency; i++ {
			wg.Go(func() {
				w.executeDBJobs(ctx, "LISTEN llm_jobs_available", keysFromMap(handlerMap), handlerMap)
			})
		}
	default:
		log.Fatal("role must be poller, worker, or llm_worker")
	}

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	log.Println("Shutdown signal received")

	cancel()

	// Forced shutdown after timeout
	const maxShutdown = 10 * time.Second
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers shut down cleanly")
	case <-time.After(maxShutdown):
		log.Println("Shutdown timed out, forcing exit")
	}

	log.Println("Worker stopped")
}

func keysFromMap(myMap map[string]JobHandler) []string {
	keys := make([]string, 0, len(myMap))

	for k := range myMap {
		keys = append(keys, k)
	}

	return keys
}
