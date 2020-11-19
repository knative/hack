#!/usr/bin/env bash

if [ "$1" != "" ]; then
  OUTPUT_FILE="$1"
else
  OUTPUT_FILE="${ARTIFACTS:-$(mktemp -d)}/bench-result.txt"
fi

echo "Output will be at $OUTPUT_FILE"

# Run all microbenchmarks
go clean
go test -bench=. -benchmem -run="^$" -v ./...   >> "$OUTPUT_FILE" || exit

