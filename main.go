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
)

const cdxAPIURL = "http://web.archive.org/cdx/search/cdx"

// Structure to hold a part of the CDX API response
// urlkey, timestamp, original, mimetype, statuscode, digest, length
// We only care about timestamp (index 1) and original (index 2) for status 200
type SnapshotEntry []interface{}

func main() {
	flag.Parse()
	urlsToCheck := flag.Args()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 { // Check if data is being piped
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
		fmt.Println("Usage:")
		fmt.Println("  timetraveller <url1> [url2 ...]")
		fmt.Println("  echo <url> | timetraveller")
		fmt.Println("  cat list_of_urls.txt | timetraveller")
		os.Exit(1)
	}

	httpClient := &http.Client{}

	for _, u := range urlsToCheck {
		fmt.Printf("Checking: %s\n", u)
		processURL(httpClient, u)
		fmt.Println() // Add a blank line for readability
	}
}

func processURL(client *http.Client, targetURL string) {
	// Construct the CDX API URL
	apiURL, err := url.Parse(cdxAPIURL)
	if err != nil {
		log.Printf("Error parsing base API URL: %v", err)
		fmt.Println("  Status: error constructing API URL")
		return
	}

	query := apiURL.Query()
	query.Set("url", targetURL)
	query.Set("output", "json")
	query.Set("filter", "statuscode:200") // Filter for successful snapshots
	apiURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v", targetURL, err)
		fmt.Println("  Status: error creating request")
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching data for %s: %v", targetURL, err)
		fmt.Println("  Status: error fetching data")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("API request failed for %s. Status: %s, Body: %s", targetURL, resp.Status, string(bodyBytes))
		fmt.Printf("  Status: API error (%s)\n", resp.Status)
		return
	}

	var cdxResponse [][]interface{} // The response is a JSON array of arrays
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&cdxResponse); err != nil {
		// Handle cases where the response might not be a valid JSON array,
		// e.g. empty response for a URL never archived or malformed.
		if err == io.EOF { // Empty response is not an error, just means no snapshots
			fmt.Println("  Status: not found")
			return
		}
		// Attempt to read the body to see if it's an error message from Wayback
		// Reset reader and try to read as plain text if JSON parsing fails
		// This part is tricky because the body might have been partially consumed.
		// For simplicity, we'll just log the JSON parsing error.
		log.Printf("Error decoding JSON for %s: %v", targetURL, err)
		fmt.Println("  Status: error decoding response")
		return
	}

	// The first element of cdxResponse is the header row, e.g. ["urlkey","timestamp","original","mimetype","statuscode","digest","length"]
	// We need to skip it if it exists and the response is not empty.
	var snapshots []SnapshotEntry
	if len(cdxResponse) > 1 {
		for _, entry := range cdxResponse[1:] { // Skip header row
			snapshots = append(snapshots, SnapshotEntry(entry))
		}
	}

	snapshotCount := len(snapshots)

	if snapshotCount > 0 {
		fmt.Println("  Status: found")
		fmt.Printf("  Snapshots: %d\n", snapshotCount)

		// The first snapshot in the filtered list (cdxResponse[1]) should be the oldest
		// because CDX server sorts by timestamp by default.
		oldestEntry := snapshots[0] // This is cdxResponse[1] effectively

		// Ensure the entry has enough elements before trying to access them
		// Indexes: 1 for timestamp, 2 for original URL
		if len(oldestEntry) > 2 {
			timestamp, tsOk := oldestEntry[1].(string)
			originalURL, origOk := oldestEntry[2].(string)

			if tsOk && origOk {
				waybackURL := fmt.Sprintf("http://web.archive.org/web/%s/%s", timestamp, originalURL)
				fmt.Printf("  Oldest: %s\n", waybackURL)
			} else {
				log.Printf("Error parsing oldest snapshot data for %s: Timestamp or Original URL missing or not strings. Entry: %+v", targetURL, oldestEntry)
				fmt.Println("  Oldest: could not determine (error parsing snapshot data)")
			}
		} else {
			log.Printf("Error parsing oldest snapshot data for %s: Not enough fields in entry. Entry: %+v", targetURL, oldestEntry)
			fmt.Println("  Oldest: could not determine (not enough fields in snapshot data)")
		}

	} else {
		// This case might also be hit if cdxResponse was empty or only had a header
		fmt.Println("  Status: not found")
	}
}
