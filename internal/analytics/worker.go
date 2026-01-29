package analytics

import (
	"context"
	"log"
	"sync"

	"url-shortener/internal/repository"
)

type Worker struct {
	store      *repository.PostgresStore
	Events     chan string
	ctx        context.Context
	cancel     context.CancelFunc
	numWorkers int
	wg         *sync.WaitGroup
}

func CreateWorker(store *repository.PostgresStore, numWorkers int) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	w := new(Worker)
	w.store = store
	w.Events = make(chan string)
	w.ctx = ctx
	w.cancel = cancel
	w.numWorkers = numWorkers
	w.wg = nil
	return w
}

func (w *Worker) Close() {
	w.cancel()
	if w.wg != nil {
		w.wg.Wait()
	}
}

func (w *Worker) RunWorker() {
	w.wg = new(sync.WaitGroup)
	log.Printf("Starting analytics worker pool with %d workers", w.numWorkers)
	var shortUrl string
	for i := 0; i < w.numWorkers; i++ {
		w.wg.Go(func() {
			for {
				select {
				case <-w.ctx.Done():
					return
				case shortUrl = <-w.Events:
					err := w.store.IncrementHits(shortUrl, w.ctx)
					if err != nil {
						log.Printf("Failed to track click for %s: %v", shortUrl, err)
					}
				}
			}
		})
	}
	log.Println("Analytics worker pool started")

}

func (w *Worker) TrackHit(url string) {
	select {
	case <-w.ctx.Done():
		return
	case w.Events <- url:
		return
	default:
		log.Println("Something went wrong when queuing work")
	}
}
