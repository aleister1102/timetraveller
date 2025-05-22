# TimeTraveller

TimeTraveller is a command-line tool to interact with the Wayback Machine (archive.org) CDX API. It allows you to check for archived snapshots of one or more URLs, retrieve the count of available snapshots, and get a link to the oldest or newest one.

## Features

- Check single or multiple URLs.
- Accept URLs from command-line arguments or piped via stdin.
- Display count of snapshots and link to the **oldest or newest** snapshot for found URLs.
- Customizable number of concurrent workers (threads).
- Customizable timeout for HTTP requests.
- Optional delay between requests to be polite to the API.
- Filter out "not found" and error results.
- Colored output for easy readability.

## Installation

### Prerequisites

- Go (version 1.16 or later recommended).

### Building from source

1.  Clone this repository or download the source code.
2.  Navigate to the `timetraveller` directory:
    ```bash
    cd timetraveller
    ```
3.  Build the executable:
    ```bash
    go build
    ```
    This will create a `timetraveller` (or `timetraveller.exe` on Windows) executable in the current directory.


### Install from GitHub

```shell
go install github.com/aleister1102/timetraveller
```

## Usage

```
./timetraveller [options] <url1> [url2 ...]
```

Or, using pipe:

```bash
echo "example.com" | ./timetraveller [options]
cat list_of_urls.txt | ./timetraveller [options]
```

(Where `list_of_urls.txt` contains one URL per line)

### Options

```
  -d int
    	Delay in milliseconds between each request sent by a worker (default 0)
  -latest
    	Get the latest snapshot instead of the oldest
  -no-err
    	Filter out 'not found' and error results
  -t int
    	Number of concurrent goroutines (threads) (default 10)
  -to int
    	Timeout for each HTTP request in milliseconds (default 10000)
```

### Output Format

The tool uses colored prefixes to indicate the status:

-   `[+] <URL> - Snapshots: <count> - Oldest: <link_to_snapshot>` (Green: URL found with snapshots, shows oldest by default)
-   `[+] <URL> - Snapshots: <count> - Latest: <link_to_snapshot>` (Green: URL found with snapshots, shows latest if `-latest` is used)
-   `[-] <URL>` (Yellow: URL not found in archive or no snapshots with HTTP 200)
-   `[!] <URL> - <error_details>` (Red: An error occurred while processing the URL)

If the `-no-err` flag is used, only `[+]` results will be shown.

### Examples

1.  **Check a single URL (gets oldest snapshot by default):**
    ```bash
    ./timetraveller google.com
    ```

2.  **Check a single URL and get the latest snapshot:**
    ```bash
    ./timetraveller -latest google.com
    ```

3.  **Check multiple URLs with 5 workers, a 2-second timeout, and get latest snapshots:**
    ```bash
    ./timetraveller -t 5 -to 2000 -latest google.com example.com
    ```

4.  **Check URLs from a file, with a 500ms delay, get oldest, and hide 'not found'/'error' results:**
    ```bash
    cat my_urls.txt | ./timetraveller -d 500 -no-err
    ```

5.  **Check a URL for timeout error (verbose output):
    ```bash
    ./timetraveller -to 500 non_existent_domain_for_timeout.com
    ```
    Output might be:
    `[!] non_existent_domain_for_timeout.com - context deadline exceeded (Client.Timeout exceeded while awaiting headers)`

## Contributing

Feel free to open issues or submit pull requests if you have suggestions or find bugs. 