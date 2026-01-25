package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"url-shortener/internal/analytics"
	"url-shortener/internal/api/handlers"
	"url-shortener/internal/api/routes"
	"url-shortener/internal/caching"
	"url-shortener/internal/repository"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("Initializing backend")
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Error loading .env file")
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
	defer store.Close()
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
	defer redisStore.Close()

	analyticsWorker := analytics.CreateWorker(store)
	go analyticsWorker.RunWorker()
	defer analyticsWorker.Close()

	handler := handlers.NewHandler(store, analyticsWorker, redisStore)
	router := routes.NewRouter(handler)
	log.Println("Initialized backend")

	port := ":8080"
	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}
}
