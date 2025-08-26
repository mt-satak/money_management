#!/bin/bash
# Go code formatting script for pre-commit hook

for file in "$@"; do
    gofmt -w "$file"
done
