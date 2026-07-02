#!/bin/sh
set -e

echo "Building speedtest..."
go build -o speedtest .

echo "Installing to /usr/local/bin/speedtest..."
sudo mv speedtest /usr/local/bin/speedtest

echo "Done. Run 'speedtest -h' to get started."
