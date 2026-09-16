#!/bin/sh
# Runs the NVIDIA Management Library (NVML) ABI test against the fake library. On Linux it runs
# directly; elsewhere it uses Docker with the official Go image.
set -e
cd "$(dirname "$0")/.."
if [ "$(uname -s)" = Linux ]; then
  exec go test -run TestFakeNVML -v ./internal/provider/nvidia/
fi
exec docker run --rm -v "$PWD":/src -w /src golang:1.25 sh -c 'go test -run TestFakeNVML -v ./internal/provider/nvidia/'
