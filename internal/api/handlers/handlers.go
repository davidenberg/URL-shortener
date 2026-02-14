package handlers

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"url-shortener/internal/models"

	"github.com/jxskiss/base62"
	"github.com/redis/go-redis/v9"
)

type URLStorage interface {
	GetURL(shortenedURL string, ctx context.Context) (string, error)
	StoreURL(shortenedURL string, originalURL string, ctx context.Context) error
	GetStatsByURL(shortenedURL string, ctx context.Context) (error, time.Time, int)
}

type AnalyticsTracker interface {
	TrackHit(url string)
}

type CacheStorage interface {
	Get(ctx context.Context, shortenedURL string) (string, error)
	Save(ctx context.Context, shortenedURL string, ogURL string, ttl time.Duration) error
}

type GenerateUrlHandler struct {
	psStore         URLStorage
	analyticsWorker AnalyticsTracker
	Redis           CacheStorage
	baseDomain      string
}

func NewHandler(s URLStorage, w AnalyticsTracker, r CacheStorage, baseDomain string) *GenerateUrlHandler {
	handler := new(GenerateUrlHandler)
	handler.psStore = s
	handler.analyticsWorker = w
	handler.Redis = r
	handler.baseDomain = baseDomain
	return handler
}

func generateShortURL(url string, len int) string {
	h := sha1.New()
	h.Write([]byte(url))
	ret := base62.EncodeToString(h.Sum(nil))
	return ret[:len]
}

func (h *GenerateUrlHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			http.Error(w, "Content-Type is not application/json", http.StatusUnsupportedMediaType)
			return
		}
	} else {
		http.Error(w, "Content-Type is not application/json", http.StatusBadRequest)
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
		_, err = h.psStore.GetURL(shortened_url, r.Context())
		if err != nil {
			http.Error(w, "Failed to generate short URL", http.StatusBadRequest)
			return
		}
	}
	resp := models.ShortenResponse{
		ShortURL: fmt.Sprintf("%s/urls/%s", h.baseDomain, shortened_url),
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

	originalURL, err := h.Redis.Get(r.Context(), shortenedURL)

	if err == redis.Nil {
		log.Printf("Cache miss %s", originalURL)

		originalURL, err = h.psStore.GetURL(shortenedURL, r.Context())
		if err == sql.ErrNoRows {
			log.Println(err)
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Println(err)
			http.Error(w, "URL not found", http.StatusInternalServerError)
			return
		}
		err = h.Redis.Save(r.Context(), shortenedURL, originalURL, time.Hour)
		if err != nil {
			log.Printf("Failed to cache URL %s, %v", shortenedURL, err)
		}
	} else if err != nil {
		log.Printf("Redis error %v", err)
		originalURL, err = h.psStore.GetURL(shortenedURL, r.Context())
		if err != nil {
			log.Println(err)
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}
	} else {
		log.Println("Cache hit")
	}

	h.analyticsWorker.TrackHit(shortenedURL)

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func (h *GenerateUrlHandler) GetURLStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	shortenedURL := strings.TrimPrefix(r.URL.Path, "/urls/stats/")
	err, time, hits := h.psStore.GetStatsByURL(shortenedURL, r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	resp := models.StatsResponse{
		CreationTime: fmt.Sprintf("%v", time),
		Hits:         hits,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
