#!/bin/bash

# Color definitions
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Use echo/printf consistently for environments that may not support -e properly
echo -e "${BLUE}==========================================${NC}"
echo -e "${BLUE}   Lumiwave Protocol Auto-Deploy System   ${NC}"
echo -e "${BLUE}==========================================${NC}"

# Variable definitions
COSMOVISOR_BIN="$HOME/go/bin/cosmovisor"
NODE_HOME="$HOME/.lumiwave-protocol"
LOG_DIR="$HOME/.lumiwave-protocol-log"

# 1. Stop existing processes
echo -e "${YELLOW}[1/6] Stopping existing processes...${NC}"
pkill cosmovisor || true
pkill lumiwave-protocold || true
sleep 2

# 2. Git pull & build
echo -e "${YELLOW}[2-3/6] Updating and building...${NC}"
git pull
ignite chain build

# 4. Reset data and configuration (optional)
# [Fix] Use standard comparison operator '=' for sh/bash compatibility
if [ "$1" = "reset" ]; then
    echo -e "${RED}[4/6] !!! RESET MODE ACTIVATED !!!${NC}"
    rm -rf "$NODE_HOME"
    
    # [Fix] Removed --force flag to match Ignite v29 behavior
    # Since the directory was already removed above, this will run without errors
    ignite chain init
    
    # Create required directories for Cosmovisor
    mkdir -p "$NODE_HOME/data"
    mkdir -p "$NODE_HOME/cosmovisor/genesis/bin"
    mkdir -p "$NODE_HOME/cosmovisor/upgrades"
    
    echo -e "${GREEN}>> Initialization completed.${NC}"
else
    echo -e "${GREEN}[4/6] Skipping reset. Ensuring data directory exists...${NC}"
    mkdir -p "$NODE_HOME/data"
fi

# 5. Sync binary to Cosmovisor genesis directory
echo -e "${YELLOW}[5/6] Syncing binary to Cosmovisor genesis...${NC}"
cp ~/go/bin/lumiwave-protocold "$NODE_HOME/cosmovisor/genesis/bin/"

# 6. Start the node
echo -e "${YELLOW}[6/6] Starting Cosmovisor in background...${NC}"
mkdir -p "$LOG_DIR"
CURRENT_LOG="$LOG_DIR/lumiwave_$(date +%Y%m%d).log"

export DAEMON_NAME="lumiwave-protocold"
export DAEMON_HOME="$NODE_HOME"
export DAEMON_ALLOW_AUTOMATIC_RESTART="true"

# [Fix] Explicitly specify --home to avoid usage errors
nohup "$COSMOVISOR_BIN" run start --home "$NODE_HOME" >> "$CURRENT_LOG" 2>&1 &

echo -e "${BLUE}------------------------------------------${NC}"
echo -e "${GREEN}✅ Deployment Finished!${NC}"
echo -e "${BLUE}▶ Log Location: ${NC}$CURRENT_LOG"
echo -e "${BLUE}▶ To view logs: ${NC}tail -f $CURRENT_LOG"
echo -e "${BLUE}==========================================${NC}"
