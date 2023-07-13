# Makefile
# Build httpheadershell
# By J. Stuart McMurray
# Created 20230714
# Last Modified 20230714

httpheadershell: check *.go
	go build -v -trimpath -ldflags "-w -s"

.PHONY: check
check: *.go
	go test
	go vet
	staticcheck
