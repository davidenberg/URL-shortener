package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"url-shortener/internal/analytics"
	"url-shortener/internal/api/handlers"
	"url-shortener/internal/api/routes"
	"url-shortener/internal/caching"
	"url-shortener/internal/repository"

	"github.com/joho/godotenv"
)

type cleanupStruct struct {
	store           *repository.PostgresStore
	redisStore      *caching.RedisStore
	analyticsWorker *analytics.Worker
}

func cleanUp(cleanup *cleanupStruct) {
	if cleanup.store != nil {
		log.Println("Cleaning up Postgres store")
		cleanup.store.Close()
	}
	if cleanup.redisStore != nil {
		log.Println("Cleaning up Redis store")
		cleanup.redisStore.Close()
	}
	if cleanup.analyticsWorker != nil {
		log.Println("Analytics worker pool shutting down")
		cleanup.analyticsWorker.Close()
		log.Println("Analytics worker pool shut down")
	}
	os.Exit(0)
}

func main() {
	cleanup := &cleanupStruct{nil, nil, nil}
	defer cleanUp(cleanup)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("Caught signal %v, exiting", sig)
			cleanUp(cleanup)
		}
	}()

	log.Println("Initializing backend")
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatalln("No database defined")
	}
	ctx := context.Background()
	store, err := repository.NewPostgresStore(databaseURL, ctx)
	if err != nil {
		log.Fatalf("Could not connect to DB: %v", databaseURL)
	}
	cleanup.store = store

	err = store.InitPostgresStore(ctx)
	if err != nil {
		log.Fatalf("Could not initialize DB: %v", databaseURL)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalln("No redis address defined")
	}
	redisStore, err := caching.NewRedisStore(redisAddr)
	if err != nil {
		log.Fatalf("Could not connect to Redis Cache: %v", err)
	}
	cleanup.redisStore = redisStore

	var numWorkers int
	numWorkers, err = strconv.Atoi(os.Getenv("ANALYTICS_WORKER_NUM"))
	if err != nil {
		numWorkers = 10
	}
	analyticsWorker := analytics.CreateWorker(store, numWorkers)
	go analyticsWorker.RunWorker()
	cleanup.analyticsWorker = analyticsWorker

	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		log.Fatalf("Please define BASE_DOMAIN env variable")
	}

	handler := handlers.NewHandler(store, analyticsWorker, redisStore, baseDomain)
	router := routes.NewRouter(handler)
	log.Println("Initialized backend")

	port := ":8080"
	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}
}
