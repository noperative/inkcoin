#!/bin/bash
# Use to run test suite

echo "Setting up server"
x-terminal-emulator -e go run ./proj1-server/server.go -c ./proj1-server/config.json

echo "simple-paths-absolute"
echo "Should draw and delete simple, absolute paths (L, H, V)"
x-terminal-emulator -e go run ./test-apps/simple-paths-absolute.go

echo "simple-overlap-absolute"
echo "Should trigger ShapeOverlapError using simple, absolute paths (L, H, V)"
x-terminal-emulator -e go run ./test-apps/simple-overlap-absolute-1.go
x-terminal-emulator -e go run ./test-apps/simple-overlap-absolute-2.go

echo "simple-overlap-relative"
echo "Should trigger ShapeOverlapError using simple, relative paths (l, h, v)"
x-terminal-emulator -e go run ./test-apps/simple-overlap-relative-1.go
x-terminal-emulator -e go run ./test-apps/simple-overlap-relative-2.go
