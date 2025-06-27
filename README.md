# 🕰️ TimeTraveller

TimeTraveller is a command-line tool written in Go to interact with the Wayback Machine (archive.org) CDX API. It allows you to check for archived snapshots of one or more URLs, retrieve the count of available snapshots, and get a link to the oldest or newest one.

## ✨ Features

-   **Check single or multiple URLs**: Pass URLs directly as arguments.
-   **Piped Input**: Accepts a list of URLs from stdin, perfect for chaining with other tools.
-   **Oldest or Latest**: Retrieve either the very first or the most recent snapshot.
-   **Concurrency**: Use multiple goroutines (threads) to process URLs in parallel, making it fast.
-   **Resilience**: Automatically retries on network errors or server-side issues (like 429s or 5xx) with an exponential backoff strategy.
-   **Filtering**: Option to hide "not found" and error messages to only show successful results.
-   **File Output**: Save all found snapshot URLs directly to a file.
-   **Colored Output**: Status indicators are color-coded for quick and easy visual parsing.

## 🛠️ Installation

### Prerequisites

-   Go (version 1.18 or later is recommended).

### Building from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/your-username/timetraveller.git
    cd timetraveller
    ```
2.  Build the executable:
    ```bash
    go build .
    ```
    This creates a `timetraveller` (or `timetraveller.exe` on Windows) executable in the directory.

## 🚀 Usage

You can pass URLs as command-line arguments or pipe them from another command's output.

**Basic Syntax:**
```bash
./timetraveller [OPTIONS] [url1] [url2]...
```

**Piping from a file:**
```bash
cat list_of_urls.txt | ./timetraveller [OPTIONS]
```

### ⚙️ Options

| Flag      | Description                                                    | Default |
|-----------|----------------------------------------------------------------|---------|
| `-t`      | Number of concurrent goroutines (threads) to use.              | `10`    |
| `-to`     | Timeout for each HTTP request in milliseconds.                 | `60000` |
| `-d`      | Delay in milliseconds between each request sent by a worker.   | `0`     |
| `-latest` | Get the latest snapshot instead of the oldest.                 | `false` |
| `-no-err` | Filter out 'not found' and error results from the output.      | `false` |
| `-o`      | File to write found snapshot URLs to.                          | `""`    |


### 🎨 Output Format

The tool uses colored prefixes to indicate the status of each URL:

-   `[+]` (Green): A snapshot was successfully found.
-   `[-]` (Yellow): The URL was not found in the archive or had no valid snapshots.
-   `[!]` (Red): An error occurred during processing. This could be a network issue or an API error after multiple retries.

### 📝 Examples

1.  **Check a single URL for its oldest snapshot:**
    ```bash
    ./timetraveller example.com
    ```

2.  **Check a URL for its latest snapshot:**
    ```bash
    ./timetraveller -latest example.com
    ```

3.  **Check multiple URLs with 20 workers and a 10-second timeout:**
    ```bash
    ./timetraveller -t 20 -to 10000 google.com github.com
    ```

4.  **Check URLs from a file and save the found snapshots to `found.txt`:**
    ```bash
    cat my_urls.txt | ./timetraveller -o found.txt
    ```

5.  **Check URLs from a file, hide errors, and use a 500ms delay between requests:**
    ```bash
    cat my_urls.txt | ./timetraveller -no-err -d 500
    ```

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/your-username/timetraveller/issues). 
