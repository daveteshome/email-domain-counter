# Email domain counter

This is a small Go CLI tool that reads a CSV file of customer data, extracts the email domains, and shows how many customers belong to each domain. The output is sorted by **count (descending)** then **domain (ascending)**., either printed to the console or written to a file.

---

## Features

- Command-line interface with clear flags  
- Gracefully handles missing or malformed rows (bad rows counted in stats)  
- Domain validation with two modes: strict or allow single-label domains (`user@corp`)  
- Deterministic sort order: highest count first, ties broken alphabetically  
- Efficient on large inputs
- Unified CSV output format for both stdout and file export  
- Comprehensive test coverage and a performance benchmark  

---

## Installation & Setup

Requirements:
- Go 1.21 or newer

Clone and build:

```sh
git clone https://github.com/daveteshome/email-domain-counter
cd email-domain-counter
go build ./...


go run . -path="customerimporter/testdata/benchmark10k.csv" -out "result.csv"
```

## Usage

```sh
Usage: importer -path=<file> [-out=<file>] [-email-header=<name>] [--allow-single-label-domain]

Flags:
  -path string
        Path to the file with customer data (required)
  -out string
        Optional: output file path (stdout if empty)
  -email-header string
        Email column header (case-insensitive, default "email")
  -allow-single-label-domain
        Accept domains without a dot (e.g., user@corp)

Examples

# Print to console
go run .  -path ./customerimporter/testdata/benchmark10k.csv

# Save to a file
go run .  -path ./customerimporter/testdata/benchmark10k.csv -out ./result.csv

# Show help
go run . -h

```
## Example output

When you run the tool with a sample dataset, you will see a summary log like this:
```sh
2025/09/24 16:58:21 INFO summary file=customerimporter/testdata/benchmark10k.csv total_rows=10000 bad_rows=0 unique_domains=501 sorted="count desc, domain asc" single_label_allowed=false
```
This output shows that the program processed benchmark10k.csv, found a total of 10,000 rows, no bad rows, and 501 unique domains, and wrote the results to result.csv.

## Testing & Benchmarking

```sh
Run all tests:

go test ./... -v

Run benchmarks:

go test ./customerimporter -bench . -benchmem
```

## Integration (Smoke) Test

In addition to unit tests, there is an optional end-to-end smoke test that runs the full CLI using a real dataset.

⚠️ **Important**: This test is written to work strictly with the `./customers.csv` file (the one provided with the task).  
If you want to use a different input file, you will need to update the `wantSnippets` in `cli_smoke_test.go` to match the expected summary (e.g., total rows, bad rows, unique domains).

### Running the integration test

The integration test is guarded by a build tag so it doesn’t run with normal `go test`.  
To run it explicitly:

```sh
go test -tags e2e -v
```

## Makefile Shortcuts

Instead of long commands, you can use:

- `make build` – build the binary  
- `make run` – run with -> `benchmark10k.csv` and post to `result.csv`
- `make test` – run all unit tests  
- `make bench` – run benchmarks  using `benchmark10k.csv` 
- `make smoke` – run the integration test with `customers.csv`  


## Project Structure
```sh

|__ main.go 
|__ Makefile      
|__ customerimporter/      
|   |__ importer.go
|   |__ importer_test.go
|   |__ testdata/
|       |__ benchmark1m.csv  #used for benchmark
|__ exporter/                
|    |__ exporter.go
|    |__ exporter_test.go
|__  cli_smoke_test.go 
|__  customers.csv  # used for intergation (smoke) test
|__ .gitignore
|__  go.mod
|__  README.md
```

## Performance Notes

This project was written with large datasets in mind (up to 1M+ rows). A few implementation details help improve throughput and reduce memory pressure:

- **Buffered CSV Reader**: Uses `bufio.NewReaderSize` with a 256 KB buffer to reduce syscalls on large files.  
- **Record Reuse**: `csv.Reader.ReuseRecord = true` ensures slices are reused instead of allocated per row, minimizing GC overhead.  
- **Pre-sized Map**: The domain frequency map is allocated with a heuristic capacity (`file size / ~40 bytes per row`), reducing expensive rehashing during large imports.  
- **Zero-copy Domain Extraction**: Domains are sliced directly from the email string when possible, avoiding allocations unless case-folding is required.  
- **Sorting**: `sort.SliceStable` is used with a clear deterministic rule (count ↓, domain ↑), ensuring consistent results across runs.  

On a 1M-row dataset (~40 MB), the importer runs in a few hundred milliseconds on a modern laptop (~160–190 MB/s in benchmarks).


## Future Improvements

- Add IDN/Punycode support for internationalized domains
- Add a custom delimiter flag (e.g. -sep=";")
- Stream results instead of keeping all counts in memory for very large files
- Configurable log level (verbose/debug vs silent)
- Add a cmd/ package and move the CLI there when the command set grows