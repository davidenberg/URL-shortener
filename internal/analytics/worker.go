package analytics

import (
	"context"
	"log"

	"url-shortener/internal/repository"

	"golang.org/x/sync/errgroup"
)

type Worker struct {
	store      *repository.PostgresStore
	events     chan string
	ctx        context.Context
	cancel     context.CancelFunc
	numWorkers int
}

func CreateWorker(store *repository.PostgresStore, numWorkers int) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	w := new(Worker)
	w.store = store
	w.events = make(chan string)
	w.ctx = ctx
	w.cancel = cancel
	w.numWorkers = numWorkers
	return w
}

func (w *Worker) Close() {
	w.cancel()
	close(w.events)
}

func (w *Worker) RunWorker() {
	g, ctx := errgroup.WithContext(w.ctx)
	log.Printf("Starting analytics worker pool with %d workers", w.numWorkers)
	var shortUrl string
	for i := 0; i < w.numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case shortUrl = <-w.events:
					err := w.store.IncrementHits(shortUrl, w.ctx)
					if err != nil {
						log.Printf("Failed to track click for %s: %v", shortUrl, err)
					}
				}
			}
		})
	}
	log.Println("Analytics worker pool started")
	err := g.Wait()
	if err != nil {
		log.Printf("Analytics worker pool shut down by error: %v", err)
	} else {
		log.Printf("Analytics worker pool shut down gracefully")
	}
}

func (w *Worker) TrackHit(url string) {
	select {
	case <-w.ctx.Done():
		return
	case w.events <- url:
		return
	default:
		log.Println("Something went wrong when queuing work")
	}
}
