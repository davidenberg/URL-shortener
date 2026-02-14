package routes

import (
	"net/http"

	"url-shortener/internal/api/handlers"
)

func NewRouter(h *handlers.GenerateUrlHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /urls/stats/{shortened_url}", h.GetURLStatistics)
	mux.HandleFunc("GET /urls/{shortened_url}", h.Redirect)
	mux.HandleFunc("POST /urls", h.ShortenURL)

	return mux
}
