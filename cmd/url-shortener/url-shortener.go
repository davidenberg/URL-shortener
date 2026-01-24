package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"personal.davidenberg.fi/url-shortener/internal/api/handlers"
	"personal.davidenberg.fi/url-shortener/internal/api/routes"
	"personal.davidenberg.fi/url-shortener/internal/repository"
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
	handler := handlers.NewHandler(store)
	router := routes.NewRouter(handler)
	log.Println("Initialized backend")

	port := ":8080"
	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}
}
