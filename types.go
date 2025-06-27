package main

const (
	cdxAPIURL = "https://web.archive.org/cdx/search/cdx"

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