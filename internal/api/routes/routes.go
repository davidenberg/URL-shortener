package routes

import (
	"net/http"

	"url-shortener/internal/api/handlers"
)

func NewRouter(h *handlers.GenerateUrlHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/urls/stats/{shortened_url}", h.GetURLStatistics)
	mux.HandleFunc("/urls/{shortened_url}", h.Redirect)
	mux.HandleFunc("/urls", h.ShortenURL)

	return mux
}
