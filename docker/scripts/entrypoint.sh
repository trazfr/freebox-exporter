#!/bin/sh

TOKEN_FILE="${TOKEN_FILE:-/app/data/token.json}"
DEBUG="${DEBUG:-false}"
HOST_DETAILS="${HOST_DETAILS:-false}"
HTTP_DISCOVERY="${HTTP_DISCOVERY:-false}"
LISTEN_ADDRESS="${LISTEN_ADDRESS:-:9091}"

# Ensure data directory exists
mkdir -p "$(dirname "$TOKEN_FILE")"

# Build command arguments
ARGS=""

if [ "$DEBUG" = "true" ]; then
    ARGS="$ARGS -debug"
fi

if [ "$HOST_DETAILS" = "true" ]; then
    ARGS="$ARGS -hostDetails"
fi

if [ "$LISTEN_ADDRESS" != ":9091" ]; then
    ARGS="$ARGS -listen $LISTEN_ADDRESS"
fi

# Check if token file exists
if [ ! -f "$TOKEN_FILE" ] || [ "$HTTP_DISCOVERY" = "true" ]; then
    if [ ! -f "$TOKEN_FILE" ]; then
        echo "First run detected: token file '$TOKEN_FILE' not found. Adding -httpDiscovery flag..."
    else
        echo "HTTP_DISCOVERY enabled, using HTTP discovery..."
    fi
    echo "Running: freebox-exporter -httpDiscovery$ARGS $TOKEN_FILE"
    exec freebox-exporter -httpDiscovery $ARGS "$TOKEN_FILE"
else
    echo "Token file '$TOKEN_FILE' found. Starting normally..."
    echo "Running: freebox-exporter$ARGS $TOKEN_FILE"
    exec freebox-exporter $ARGS "$TOKEN_FILE"
fi
