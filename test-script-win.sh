echo "Setting up server"
start powershell -noexit {cd $pwd; go run ./proj1-server/server.go -c ./proj1-server/config.json}

echo "simple-paths-absolute"
echo "Should draw and delete simple, absolute paths (L, H, V)"
start powershell -noexit {go run ./test-apps/simple-paths-absolute.go}

echo "simple-overlap-absolute"
echo "Should trigger ShapeOverlapError using simple, absolute paths (L, H, V)"
start powershell -noexit {go run ./test-apps/simple-overlap-absolute-1.go}
start powershell -noexit {go run ./test-apps/simple-overlap-absolute-2.go}

echo "simple-overlap-relative"
echo "Should trigger ShapeOverlapError using simple, relative paths (l, h, v)"
start powershell -noexit {go run ./test-apps/simple-overlap-relative-1.go}
start powershell -noexit {go run ./test-apps/simple-overlap-relative-2.go}
