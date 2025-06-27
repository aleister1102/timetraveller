package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	numWorkersFlag       *int
	requestTimeoutMsFlag *int
	noErrorFilterFlag    *bool
	delayMsFlag          *int
	latestSnapshotFlag   *bool
	outputFileFlag       *string
)

func main() {
	numWorkersFlag = flag.Int("t", 10, "Number of concurrent goroutines (threads)")
	requestTimeoutMsFlag = flag.Int("to", 60000, "Timeout for each HTTP request in milliseconds")
	noErrorFilterFlag = flag.Bool("no-err", false, "Filter out 'not found' and error results")
	delayMsFlag = flag.Int("d", 0, "Delay in milliseconds between each request sent by a worker")
	latestSnapshotFlag = flag.Bool("latest", false, "Get the latest snapshot instead of the oldest")
	outputFileFlag = flag.String("o", "", "File to write found snapshot URLs to")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: timetraveller [options] <url1> [url2 ...]\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOr pipe URLs:\n")
		fmt.Fprintf(os.Stderr, "  echo <url> | timetraveller [options]\n")
		fmt.Fprintf(os.Stderr, "  cat list_of_urls.txt | timetraveller [options]\n")
	}
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
		// Banner is already printed. Now print usage.
		flag.Usage()
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
		go worker(i+1, httpClient, jobs, resultsChan, &wg, *delayMsFlag, 3, 5000)
	}

	// Send jobs
	for _, u := range urlsToCheck {
		jobs <- u
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var foundSnapshotURLs []string

	// Process and print results
	for result := range resultsChan {
		if *noErrorFilterFlag {
			if result.Error != nil {
				continue
			}
			if result.Status == "not found" {
				continue
			}
		}

		var outputLine string
		label := "Oldest:"
		if *latestSnapshotFlag {
			label = "Latest:"
		}

		if result.Error != nil {
			outputLine = fmt.Sprintf(ColorRed+"[!] %s - %v"+ColorReset,
				result.URL, result.Error)
		} else {
			switch result.Status {
			case "found":
				outputLine = fmt.Sprintf(ColorGreen+"[+] %s - Snapshots: %d - %s %s"+ColorReset,
					result.URL, result.SnapshotCount, label, result.OldestURL)
				foundSnapshotURLs = append(foundSnapshotURLs, result.OldestURL)
			case "not found":
				outputLine = fmt.Sprintf(ColorYellow+"[-] %s"+ColorReset,
					result.URL)
			default:
				outputLine = fmt.Sprintf(ColorCyan+"[i] %s - Status: %s (Unknown)"+ColorReset,
					result.URL, result.Status)
			}
		}
		fmt.Println(outputLine)
	}

	if *outputFileFlag != "" && len(foundSnapshotURLs) > 0 {
		if err := writeUrlsToFile(*outputFileFlag, foundSnapshotURLs); err != nil {
			log.Fatalf("Error writing to output file: %v", err)
		}
		fmt.Printf(ColorBlue+"\n[i] Successfully wrote %d found URLs to %s\n"+ColorReset, len(foundSnapshotURLs), *outputFileFlag)
	}
}
