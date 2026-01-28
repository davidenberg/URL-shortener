package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"url-shortener/internal/models"

	"github.com/redis/go-redis/v9"
)

type MockURLStore struct {
	data       map[string]string
	created_at map[string]time.Time
	hits       map[string]int
}

func (m *MockURLStore) GetURL(shortenedURL string, ctx context.Context) (string, error) {
	val, ok := m.data[shortenedURL]
	if !ok {
		return "", sql.ErrNoRows
	}
	return val, nil
}

func (m *MockURLStore) StoreURL(shortenedURL string, originalURL string, ctx context.Context) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	if m.created_at == nil {
		m.created_at = make(map[string]time.Time)
	}
	m.data[shortenedURL] = originalURL
	m.created_at[shortenedURL] = time.Now()
	return nil
}

func (m *MockURLStore) GetStatsByURL(shortenedURL string, ctx context.Context) (error, time.Time, int) {
	val, ok := m.created_at[shortenedURL]
	if !ok {
		return sql.ErrNoRows, time.Time{}, 0
	}

	hits, ok := m.hits[shortenedURL]
	if !ok {
		return sql.ErrNoRows, time.Time{}, 0
	}

	return nil, val, hits
}

type MockCache struct{}

func (m *MockCache) Get(ctx context.Context, shortenedURL string) (string, error) {
	return "", redis.Nil
}
func (m *MockCache) Save(ctx context.Context, shortenedURL string, ogURL string, ttl time.Duration) error {
	return nil
}

type MockTracker struct{}

func (m *MockTracker) TrackHit(url string) {}

func TestShortenURL(t *testing.T) {
	mockStore := &MockURLStore{}
	mockCache := &MockCache{}
	mockTracker := &MockTracker{}
	handler := NewHandler(mockStore, mockTracker, mockCache, "")
	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "Valid URL",
			body:           models.ShortenRequest{OriginalURL: "https://google.com"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid URL",
			body:           models.ShortenRequest{OriginalURL: "not-a-url"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Body",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var jsonBody []byte
			if tc.body != nil {
				jsonBody, _ = json.Marshal(tc.body)
			}

			req, _ := http.NewRequest("POST", "/urls", bytes.NewBuffer(jsonBody))

			rr := httptest.NewRecorder()
			handler.ShortenURL(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %v, got %v", tc.expectedStatus, rr.Code)
			}

			if tc.expectedStatus == http.StatusOK {
				var resp models.ShortenResponse
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp.ShortURL == "" {
					t.Error("expected short_url in response, got empty")
				}
			}
		})
	}
}

func TestRedirect(t *testing.T) {
	mockStore := &MockURLStore{
		data: map[string]string{"abcd1234": "https://example.com"},
	}
	mockCache := &MockCache{}
	mockTracker := &MockTracker{}
	handler := NewHandler(mockStore, mockTracker, mockCache, "")

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedLoc    string
	}{
		{
			name:           "Existing ID",
			path:           "abcd1234",
			expectedStatus: http.StatusFound,
			expectedLoc:    "https://example.com",
		},
		{
			name:           "Non-Existing ID",
			path:           "axyz9999",
			expectedStatus: http.StatusNotFound,
			expectedLoc:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.Redirect(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %v, got %v", tc.expectedStatus, rr.Code)
			}

			if tc.expectedLoc != "" {
				loc := rr.Header().Get("location")
				if loc != tc.expectedLoc {
					t.Errorf("expected location %v, got %v", tc.expectedLoc, loc)
				}
			}
		})
	}
}

func TestGetURLStatistics(t *testing.T) {
	creat_time := time.Now()

	hits := rand.Int()

	mockStore := &MockURLStore{
		data:       map[string]string{"abcd1234": "https://example.com"},
		created_at: map[string]time.Time{"abcd1234": creat_time},
		hits:       map[string]int{"abcd1234": hits},
	}
	mockCache := &MockCache{}
	mockTracker := &MockTracker{}
	handler := NewHandler(mockStore, mockTracker, mockCache, "")

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedTime   time.Time
		expectedHits   int
	}{
		{
			name:           "Existing ID",
			path:           "abcd1234",
			expectedStatus: http.StatusFound,
			expectedTime:   creat_time,
			expectedHits:   hits,
		},
		{
			name:           "Non-Existing ID",
			path:           "axyz9999",
			expectedStatus: http.StatusNotFound,
			expectedTime:   time.Time{},
			expectedHits:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			handler.Redirect(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %v, got %v", tc.expectedStatus, rr.Code)
			}
			if rr.Code == http.StatusOK {

				var resp models.StatsResponse
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp.CreationTime != tc.expectedTime.String() {
					t.Errorf("expected creation time %s, got %s", resp.CreationTime, tc.expectedTime.String())
				}

				if resp.Hits != tc.expectedHits {
					t.Errorf("expected hits %d, got %d", resp.Hits, tc.expectedHits)
				}
			}
		})
	}
}
