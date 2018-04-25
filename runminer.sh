#!/bin/bash
# Use to run a miner

LASTIFS=$IFS
IFS=$'\n'
result=($(go run misc/encrypt.go))
IFS=$LASTIFS
bash -c "${result[3]}"
