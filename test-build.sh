#!/bin/bash
# Use to make sure that everything compiles fine.
# Does not check proj1-server right now.

# Check top files individually since each have their own func main()
echo "Testing ./"
top_files=($(find ./*.go -maxdepth 0 -type f))
for f in ${top_files[@]}; do
	go test $f;
done

echo "Testing blockartlib/"
go test ./blockartlib/*.go
echo "Testing blockchain/"
go test ./blockchain/*.go
echo "Testing libminer/"
go test ./libminer/*.go
echo "Testing miner/"
go test ./miner/*.go
echo "Testing minerserver/"
go test ./minerserver/*.go
echo "Testing shapelib/"
go test ./shapelib/*.go
echo "Testing utils/"
go test ./utils/*.go
# go test ./misc/*.go
