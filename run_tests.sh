#!/bin/bash
echo
echo "add '-v' on command line for a detailed output"
echo
go mod tidy
go test $@ -count=1 ./...
