package handlers

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jxskiss/base62"
	"personal.davidenberg.fi/url-shortener/internal/models"
	"personal.davidenberg.fi/url-shortener/internal/repository"
)

type GenerateUrlHandler struct {
	psStore *repository.PostgresStore
}

func NewHandler(ps *repository.PostgresStore) *GenerateUrlHandler {
	handler := new(GenerateUrlHandler)
	handler.psStore = ps
	return handler
}

func generateShortURL(url string, len int) string {
	h := sha1.New()
	h.Write([]byte(url))
	ret := base62.EncodeToString(h.Sum(nil))
	return ret[:len]
}

func (h *GenerateUrlHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			http.Error(w, "Content-Type is not application/json", http.StatusUnsupportedMediaType)
			return
		}
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err = url.ParseRequestURI(req.OriginalURL)
	if err != nil {
		http.Error(w, "Invalid url", http.StatusBadRequest)
		return
	}

	shortened_url := generateShortURL(req.OriginalURL, models.SHORT_URL_LEN)
	err = h.psStore.StoreURL(shortened_url, req.OriginalURL, r.Context())
	if err != nil {
		http.Error(w, "Failed to generate short URL", http.StatusBadRequest)
		return
	}
	resp := models.ShortenResponse{
		ShortURL: fmt.Sprintf("personal.davidenberg.fi/%s", shortened_url),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *GenerateUrlHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	shortenedURL := strings.TrimPrefix(r.URL.Path, "/urls/")
	originalURL, err := h.psStore.GetURL(shortenedURL, r.Context())

	if err != nil {
		log.Println(err)
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}
