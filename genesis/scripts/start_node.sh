#!/bin/bash

LOG_DIR="$HOME/.lumiwave-protocol/logs"
LOG_FILE="$LOG_DIR/node.log"
LOGROTATE_CONF="/etc/logrotate.d/cosmovisor"

# Stop running processes
if pgrep -f "lumiwave-protocold" > /dev/null 2>&1; then
    echo "Stopping lumiwave-protocold..."
    pkill -f lumiwave-protocold
    sleep 2
fi

if pgrep -f "cosmovisor" > /dev/null 2>&1; then
    echo "Stopping cosmovisor..."
    pkill -f cosmovisor
    sleep 2
fi

# Force kill if still running
if pgrep -f "lumiwave-protocold" > /dev/null 2>&1; then
    echo "Force killing..."
    pkill -9 -f lumiwave-protocold
    sleep 1
fi

# Create log directory
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
    echo "Log directory created: $LOG_DIR"
fi

# Setup logrotate if not configured
if [ ! -f "$LOGROTATE_CONF" ]; then
    echo "Setting up logrotate..."
    sudo tee "$LOGROTATE_CONF" > /dev/null << EOF
$LOG_FILE {
    daily
    rotate 30
    size 100M
    missingok
    notifempty
    compress
    copytruncate
    dateext
}
EOF
    echo "Logrotate configured: $LOGROTATE_CONF"
fi

# Start node (strip ANSI color codes for clean log files)
echo "Starting node..."
cosmovisor run start 2>&1 | sed 's/\x1b\[[0-9;]*m//g' >> "$LOG_FILE" &

echo "Node started (PID: $!)"
echo "View logs: tail -f $LOG_FILE"
