package analytics

import (
	"context"
	"log"

	"personal.davidenberg.fi/url-shortener/internal/repository"
)

type Worker struct {
	store  *repository.PostgresStore
	events chan string
	ctx    context.Context
	cancel context.CancelFunc
}

func CreateWorker(store *repository.PostgresStore) *Worker {
	c := make(chan string)
	w := new(Worker)
	ctx, cancel := context.WithCancel(context.Background())
	w.store = store
	w.events = c
	w.ctx = ctx
	w.cancel = cancel
	return w
}

func (w *Worker) Close() {
	w.cancel()
	close(w.events)
}

func (w *Worker) RunWorker() {
	log.Println("Analytics worker started")
	var shortUrl string
	for {
		select {
		case <-w.ctx.Done():
			return
		case shortUrl = <-w.events:
			err := w.store.IncrementHits(shortUrl, w.ctx)
			if err != nil {
				log.Printf("Failed to track click for %s: %v", shortUrl, err)
			}
		}
	}
}

func (w *Worker) TrackHit(url string) {
	select {
	case w.events <- url:
		return
	default:
		log.Println("Something went wrong when queuing work")
	}
}
