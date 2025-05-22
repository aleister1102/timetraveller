package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	cdxAPIURL = "http://web.archive.org/cdx/search/cdx"

	// ANSI Color Codes
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// SnapshotEntry defines the structure of a single entry from CDX API (partially).
type SnapshotEntry []interface{}

// ProcessResult holds the outcome of processing a single URL.
type ProcessResult struct {
	URL           string
	Status        string // "found", "not found", "error"
	SnapshotCount int
	OldestURL     string
	Error         error // Holds any error encountered during processing
}

var (
	numWorkersFlag       *int
	requestTimeoutMsFlag *int
	noErrorFilterFlag    *bool
	delayMsFlag          *int // New flag for delay
)

func main() {
	numWorkersFlag = flag.Int("t", 10, "Number of concurrent goroutines (threads)")
	requestTimeoutMsFlag = flag.Int("to", 10000, "Timeout for each HTTP request in milliseconds")
	noErrorFilterFlag = flag.Bool("no-err", false, "Filter out 'not found' and error results")
	delayMsFlag = flag.Int("d", 0, "Delay in milliseconds between each request sent by a worker") // Default 0ms

	flag.Parse()

	urlsToCheck := flag.Args()

	// Read from stdin if no args are provided and data is piped
	stat, _ := os.Stdin.Stat()
	if len(urlsToCheck) == 0 && (stat.Mode()&os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				urlsToCheck = append(urlsToCheck, line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading from stdin: %v", err)
		}
	}

	if len(urlsToCheck) == 0 {
		fmt.Println("Usage: timetraveller [options] <url1> [url2 ...]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nOr pipe URLs:")
		fmt.Println("  echo <url> | timetraveller [options]")
		fmt.Println("  cat list_of_urls.txt | timetraveller [options]")
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(*requestTimeoutMsFlag) * time.Millisecond,
	}

	jobs := make(chan string, len(urlsToCheck))
	resultsChan := make(chan ProcessResult, len(urlsToCheck))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < *numWorkersFlag; i++ {
		wg.Add(1)
		// Pass delayMsFlag to the worker
		go worker(i+1, httpClient, jobs, resultsChan, &wg, *delayMsFlag)
	}

	// Send jobs (no delay needed here as workers will handle it)
	for _, u := range urlsToCheck {
		jobs <- u
	}
	close(jobs)

	// Collect results in a separate goroutine to close resultsChan once all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Process and print results
	for result := range resultsChan {
		// Unified check for -no-err flag
		if *noErrorFilterFlag {
			if result.Error != nil {
				continue
			}
			if result.Status == "not found" {
				continue
			}
		}

		var outputLine string

		if result.Error != nil {
			outputLine = fmt.Sprintf(ColorRed+"[!] %s - %v"+ColorReset,
				result.URL, result.Error)
		} else {
			switch result.Status {
			case "found":
				outputLine = fmt.Sprintf(ColorGreen+"[+] %s - Snapshots: %d - Oldest: %s"+ColorReset,
					result.URL, result.SnapshotCount, result.OldestURL)
			case "not found":
				outputLine = fmt.Sprintf(ColorYellow+"[-] %s"+ColorReset,
					result.URL)
			default: // Should not happen, but good for a fallback
				outputLine = fmt.Sprintf(ColorCyan+"[i] %s - Status: %s (Unknown)"+ColorReset, // Retain status for unknown case
					result.URL, result.Status)
			}
		}
		fmt.Println(outputLine)
	}
}

func worker(id int, client *http.Client, urls <-chan string, results chan<- ProcessResult, wg *sync.WaitGroup, delayMs int) {
	defer wg.Done()
	for targetURL := range urls {
		// fmt.Printf("Worker %d processing %s\n", id, targetURL) // Optional: for debugging
		results <- fetchURLData(client, targetURL)

		// Apply delay if specified
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
}

// fetchURLData processes a single URL and returns its ProcessResult.
// (Previously processURL, now returns a struct instead of printing)
func fetchURLData(client *http.Client, targetURL string) ProcessResult {
	result := ProcessResult{URL: targetURL}

	apiURL, err := url.Parse(cdxAPIURL)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Errorf("error parsing base API URL: %w", err)
		return result
	}

	query := apiURL.Query()
	query.Set("url", targetURL)
	query.Set("output", "json")
	query.Set("filter", "statuscode:200")
	apiURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Errorf("error creating request: %w", err)
		return result
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Errorf("error fetching data: %w", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for context
		result.Status = "error"
		result.Error = fmt.Errorf("API request failed. Status: %s, Body: %s", resp.Status, string(bodyBytes))
		return result
	}

	var cdxResponse [][]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&cdxResponse); err != nil {
		if err == io.EOF || (len(cdxResponse) == 0) { // Empty response or no actual data rows
			result.Status = "not found"
			return result
		}
		result.Status = "error"
		result.Error = fmt.Errorf("error decoding JSON response: %w", err)
		return result
	}

	var snapshots []SnapshotEntry
	if len(cdxResponse) > 1 { // cdxResponse[0] is header
		for _, entryData := range cdxResponse[1:] {
			snapshots = append(snapshots, SnapshotEntry(entryData))
		}
	} else if len(cdxResponse) == 1 && len(cdxResponse[0]) > 0 {
		// This case handles when API returns only header, meaning no actual snapshots for status:200
		result.Status = "not found"
		return result
	}

	snapshotCount := len(snapshots)

	if snapshotCount > 0 {
		result.Status = "found"
		result.SnapshotCount = snapshotCount

		oldestEntry := snapshots[0]
		if len(oldestEntry) > 2 { // Need at least timestamp (idx 1) and original URL (idx 2)
			timestamp, tsOk := oldestEntry[1].(string)
			originalURL, origOk := oldestEntry[2].(string)

			if tsOk && origOk {
				result.OldestURL = fmt.Sprintf("http://web.archive.org/web/%s/%s", timestamp, originalURL)
			} else {
				// Log this server-side if possible, or return a more specific error for the user.
				// For now, mark as unable to determine.
				result.OldestURL = "could not determine (error parsing snapshot data)"
				// Optionally, set result.Error here or change status.
			}
		} else {
			result.OldestURL = "could not determine (not enough fields in snapshot data)"
		}
	} else {
		result.Status = "not found"
	}
	return result
}
