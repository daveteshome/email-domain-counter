# Makefile

.PHONY: build run test bench smoke clean

build:
	go build -o bin/email-domain-counter .

run:
	go run . -path=./customerimporter/testdata/benchmark10k.csv -out result.csv

test:
	go test ./... -v

bench:
	go test ./customerimporter -bench . -benchmem

# Optional: run the e2e smoke test with customers.csv
smoke:
	go test -tags e2e -v

clean:
	go clean
	rm -rf bin
