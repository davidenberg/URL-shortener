package models

var SHORT_URL_LEN = 8

type ShortenRequest struct {
	OriginalURL string `json:"original_url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

type StatsResponse struct {
	CreationTime string `json:"creation_time"`
	Hits         int    `json:"hits"`
}
