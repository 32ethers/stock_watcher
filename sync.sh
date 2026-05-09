#!/bin/bash
cd "$(dirname "$0")"
GOOS=linux GOARCH=amd64 go build -o stock-watcher . && rsync -avz stock-watcher bwh1:~/stock_watcher/
