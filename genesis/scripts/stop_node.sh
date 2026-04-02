#!/bin/bash

echo "Stopping node..."

# Stop cosmovisor
if pgrep -f "cosmovisor" > /dev/null 2>&1; then
    pkill -f cosmovisor
    echo "cosmovisor stop requested"
fi

# Stop lumiwave-protocold
if pgrep -f "lumiwave-protocold" > /dev/null 2>&1; then
    pkill -f lumiwave-protocold
    echo "lumiwave-protocold stop requested"
fi

# Wait for shutdown (max 10 seconds)
for i in $(seq 1 10); do
    if ! pgrep -f "lumiwave-protocold" > /dev/null 2>&1; then
        echo "Node stopped successfully"
        exit 0
    fi
    sleep 1
done

# Force kill
echo "Force killing..."
pkill -9 -f lumiwave-protocold
pkill -9 -f cosmovisor
sleep 1

if ! pgrep -f "lumiwave-protocold" > /dev/null 2>&1; then
    echo "Node force killed successfully"
else
    echo "Failed to stop - manual check required"
    ps aux | grep -E "cosmovisor|lumiwave-protocold" | grep -v grep
fi
