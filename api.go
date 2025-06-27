package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// fetchURLData fetches snapshot data for a given URL from the CDX API.
// It implements retry logic with exponential backoff for network errors and rate limiting.
func fetchURLData(client *http.Client, targetURL string, latest bool, retryAttempts int, retryDelayMs int) ProcessResult {
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

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= retryAttempts; attempt++ {
		// Add exponential backoff delay before retrying
		if attempt > 0 {
			delay := time.Duration(retryDelayMs) * time.Millisecond * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}

		req, err := http.NewRequest("GET", apiURL.String(), nil)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Errorf("error creating request: %w", err)
			return result
		}

		resp, err = client.Do(req)
		if err != nil {
			lastErr = err // Network error
			if attempt < retryAttempts {
				continue
			}
			result.Status = "error"
			result.Error = fmt.Errorf("error fetching data after %d retries: %w", retryAttempts, lastErr)
			return result
		}

		// Read body to check for custom rate limit message.
		// We need to be able to re-read it if it's not a rate limit message.
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close() // Close original body
		if readErr != nil {
			result.Status = "error"
			result.Error = fmt.Errorf("error reading response body: %w", readErr)
			return result
		}
		// Restore body for subsequent reads.
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Check for retryable conditions: rate limiting or server-side errors (5xx).
		is429 := resp.StatusCode == http.StatusTooManyRequests
		is5xx := resp.StatusCode >= 500 && resp.StatusCode < 600
		isRateLimitMessage := strings.Contains(string(bodyBytes), "You have sent too many requests in a given amount of time.")

		if is429 || is5xx || isRateLimitMessage {
			if is429 || isRateLimitMessage {
				lastErr = fmt.Errorf("API request failed due to rate limiting. Status: %s", resp.Status)
			} else { // is5xx
				lastErr = fmt.Errorf("API request failed with server error. Status: %s", resp.Status)
			}

			if attempt < retryAttempts {
				continue
			}
			result.Status = "error"
			result.Error = fmt.Errorf("%w after %d retries", lastErr, retryAttempts)
			return result
		}

		// If we reach here, we have a response that is not a network error and not a rate limit.
		// Break the loop and process it.
		break
	}

	if resp == nil {
		// This can happen if all retries fail with a network error.
		result.Status = "error"
		if lastErr == nil {
			lastErr = fmt.Errorf("unknown error; no response received")
		}
		result.Error = fmt.Errorf("failed to get a response after all retries: %w", lastErr)
		return result
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		result.Status = "error"
		result.Error = fmt.Errorf("API request failed. Status: %s, Body: %s", resp.Status, string(bodyBytes))
		return result
	}

	var cdxResponse [][]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&cdxResponse); err != nil {
		if err == io.EOF || (len(cdxResponse) == 0) {
			result.Status = "not found"
			return result
		}
		result.Status = "error"
		result.Error = fmt.Errorf("error decoding JSON response: %w", err)
		return result
	}

	var snapshots []SnapshotEntry
	if len(cdxResponse) > 1 {
		for _, entryData := range cdxResponse[1:] {
			snapshots = append(snapshots, SnapshotEntry(entryData))
		}
	} else if len(cdxResponse) == 1 && len(cdxResponse[0]) > 0 {
		result.Status = "not found"
		return result
	}

	snapshotCount := len(snapshots)

	if snapshotCount > 0 {
		result.Status = "found"
		result.SnapshotCount = snapshotCount

		var chosenEntry SnapshotEntry
		if latest && len(snapshots) > 0 {
			chosenEntry = snapshots[len(snapshots)-1] // Get the last snapshot for "latest"
		} else if len(snapshots) > 0 {
			chosenEntry = snapshots[0] // Default to the first snapshot (oldest)
		} else {
			result.Status = "not found" // Should be caught earlier, but defensive
			return result
		}

		if len(chosenEntry) > 2 {
			timestamp, tsOk := chosenEntry[1].(string)
			originalURL, origOk := chosenEntry[2].(string)

			if tsOk && origOk {
				result.OldestURL = fmt.Sprintf("http://web.archive.org/web/%s/%s", timestamp, originalURL)
			} else {
				result.OldestURL = "could not determine (error parsing snapshot data)"
			}
		} else {
			result.OldestURL = "could not determine (not enough fields in snapshot data)"
		}
	} else {
		result.Status = "not found"
	}
	return result
}
