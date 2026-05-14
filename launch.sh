#!/bin/bash
set -e

CLOUDFLARED="${CLOUDFLARED:-cloudflared}"
if ! command -v "$CLOUDFLARED" >/dev/null 2>&1; then
    if [ -x "$HOME/bin/cloudflared" ]; then
        CLOUDFLARED="$HOME/bin/cloudflared"
    elif [ -x "$HOME/bin/cloudflared.exe" ]; then
        CLOUDFLARED="$HOME/bin/cloudflared.exe"
    fi
fi

if ! command -v "$CLOUDFLARED" >/dev/null 2>&1 && [ ! -x "$CLOUDFLARED" ]; then
    echo "cloudflared not found in PATH or ~/bin."
    echo "Install cloudflared or set CLOUDFLARED=/path/to/cloudflared before running this launcher."
    exit 1
fi

if [ ! -x "./server" ]; then
    echo "server binary not found. Build it first:"
    echo "go build -o server ./cmd/server"
    exit 1
fi

echo "Starting Cloudflare Quick Tunnel..."
"$CLOUDFLARED" tunnel --url http://127.0.0.1:3939 > /tmp/cloudflared.log 2>&1 &
TUNNEL_PID=$!

echo "Waiting for tunnel URL..."
TUNNEL_URL=""
for _ in $(seq 1 30); do
    TUNNEL_URL=$(grep -oE 'https://[a-zA-Z0-9-]+\.trycloudflare\.com' /tmp/cloudflared.log | head -1 || true)
    if [ -n "$TUNNEL_URL" ]; then
        break
    fi
    sleep 1
done

if [ -z "$TUNNEL_URL" ]; then
    echo "Could not detect tunnel URL. Check /tmp/cloudflared.log manually."
    cat /tmp/cloudflared.log
    kill "$TUNNEL_PID" >/dev/null 2>&1 || true
    exit 1
fi

echo ""
echo "========================================"
echo "  Tunnel URL: $TUNNEL_URL"
echo "  MCP Endpoint: $TUNNEL_URL/mcp/"
echo "========================================"
echo ""
echo "Paste this URL into ChatGPT connector:"
echo "$TUNNEL_URL/mcp/"
echo ""

CONFIG="./configs/project-brain.yml"
if [ ! -f "$CONFIG" ]; then
    CONFIG="./configs/project-brain.example.yml"
fi
./server --config "$CONFIG" --issuer-url "$TUNNEL_URL"
