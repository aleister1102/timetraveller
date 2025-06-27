package main

import (
	"net/http"
	"sync"
	"time"
)

func worker(id int, client *http.Client, urls <-chan string, results chan<- ProcessResult, wg *sync.WaitGroup, delayMs int, retryAttempts int, retryDelayMs int) {
	defer wg.Done()
	for targetURL := range urls {
		results <- fetchURLData(client, targetURL, *latestSnapshotFlag, retryAttempts, retryDelayMs)
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
}
