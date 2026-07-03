#!/bin/sh
set -e

echo "Building speedtest..."
go build -o speedtest .

if [ -n "$PREFIX" ] && [ -d "$PREFIX/bin" ]; then
	echo "Installing to $PREFIX/bin/speedtest..."
	mv speedtest "$PREFIX/bin/speedtest"
else
	echo "Installing to /usr/local/bin/speedtest..."
	sudo mv speedtest /usr/local/bin/speedtest
fi

echo "Done. Run 'speedtest -h' to get started."
