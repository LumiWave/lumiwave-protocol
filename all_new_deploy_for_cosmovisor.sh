#!/bin/bash

# 색상 정의
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# -e 옵션을 해석하지 못하는 환경을 위해 printf 또는 echo 사용 방식 통일
echo -e "${BLUE}==========================================${NC}"
echo -e "${BLUE}   Lumiwave Protocol Auto-Deploy System   ${NC}"
echo -e "${BLUE}==========================================${NC}"

# 변수 설정
COSMOVISOR_BIN="$HOME/go/bin/cosmovisor"
NODE_HOME="$HOME/.lumiwave-protocol"
LOG_DIR="$HOME/.lumiwave-protocol-log"

# 1. 프로그램 종료
echo -e "${YELLOW}[1/6] Stopping existing processes...${NC}"
pkill cosmovisor || true
pkill lumiwave-protocold || true
sleep 2

# 2. Git Pull & Build
echo -e "${YELLOW}[2-3/6] Updating and Building...${NC}"
git pull
ignite chain build

# 4. 데이터 및 설정 초기화
# [수정] 표준 비교 연산자 '=' 사용 (sh/bash 호환)
if [ "$1" = "reset" ]; then
    echo -e "${RED}[4/6] !!! RESET MODE ACTIVATED !!!${NC}"
    rm -rf "$NODE_HOME"
    
    # [수정] Ignite v29에 맞춰 --force 제거
    # 이미 위에서 rm -rf로 밀었기 때문에 에러 없이 실행됩니다.
    ignite chain init
    
    # Cosmovisor를 위한 필수 폴더 생성
    mkdir -p "$NODE_HOME/data"
    mkdir -p "$NODE_HOME/cosmovisor/genesis/bin"
    mkdir -p "$NODE_HOME/cosmovisor/upgrades"
    
    echo -e "${GREEN}>> Initialization completed.${NC}"
else
    echo -e "${GREEN}[4/6] Skipping reset. Ensuring data directory exists...${NC}"
    mkdir -p "$NODE_HOME/data"
fi

# 5. 바이너리 복사
echo -e "${YELLOW}[5/6] Syncing binary to Cosmovisor genesis...${NC}"
cp ~/go/bin/lumiwave-protocold "$NODE_HOME/cosmovisor/genesis/bin/"

# 6. 실행
echo -e "${YELLOW}[6/6] Starting Cosmovisor in background...${NC}"
mkdir -p "$LOG_DIR"
CURRENT_LOG="$LOG_DIR/lumiwave_$(date +%Y%m%d).log"

export DAEMON_NAME="lumiwave-protocold"
export DAEMON_HOME="$NODE_HOME"
export DAEMON_ALLOW_AUTOMATIC_RESTART="true"

# [수정] start 명령어 뒤에 --home을 명시하여 Usage 에러 방지
nohup "$COSMOVISOR_BIN" run start --home "$NODE_HOME" >> "$CURRENT_LOG" 2>&1 &

echo -e "${BLUE}------------------------------------------${NC}"
echo -e "${GREEN}✅ Deployment Finished!${NC}"
echo -e "${BLUE}▶ Log Location: ${NC}$CURRENT_LOG"
echo -e "${BLUE}▶ To view logs: ${NC}tail -f $CURRENT_LOG"
echo -e "${BLUE}==========================================${NC}"