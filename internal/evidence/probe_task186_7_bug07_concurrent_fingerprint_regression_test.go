package evidence

import (
	"sync"
	"testing"

	"task186-namemerge/internal/model"
)

func TestTask186Bug07_ConcurrentPublicationFingerprintStable(t *testing.T) {
	const workers = 20
	publication := model.Publication{Title: "Species Plantarum", Authors: "Linnaeus, C.", Journal: "Sp. Pl."}
	want := FingerprintPublication(publication)
	start := make(chan struct{})
	results := make(chan string, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			results <- FingerprintPublication(publication)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for got := range results {
		if got != want {
			t.Fatalf("concurrent fingerprint = %q, want %q", got, want)
		}
	}
}
