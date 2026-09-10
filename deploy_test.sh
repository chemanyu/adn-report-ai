#!/usr/bin/env bash
set -euo pipefail

# Match the deployment conventions used by ../MCP/deploy_test.sh.
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
REMOTE="root@172.16.3.34"
REMOTE_DIR="/data/adn-report-ai"
BUILD_DIR="$ROOT_DIR/.cache/deploy"
NPM="${NPM:-npm}"
STAGE=false
SYNC_CONFIG=false
for argument in "$@"; do
    case "$argument" in
        --stage) STAGE=true ;;
        --sync-config) SYNC_CONFIG=true ;;
        *) echo "用法：$0 [--stage] [--sync-config]" >&2; exit 2 ;;
    esac
done
source "$ROOT_DIR/scripts/node_env.sh"
adn_setup_node

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10)
SCP=(scp -o BatchMode=yes -o ConnectTimeout=10)
mkdir -p "$BUILD_DIR"
chmod 700 "$BUILD_DIR"
export GOCACHE="$ROOT_DIR/.cache/go-build"

REMOTE_CONFIG=false
if "${SSH[@]}" "$REMOTE" "test -f '$REMOTE_DIR/etc/config.yaml'"; then
    REMOTE_CONFIG=true
fi
if [[ "$REMOTE_CONFIG" == false || "$SYNC_CONFIG" == true ]]; then
    install -m 600 "$ROOT_DIR/etc/config.yaml" "$BUILD_DIR/config.yaml"
fi

echo "构建 Vue 与 Linux amd64 程序..."
"$NPM" --prefix web ci --cache "$ROOT_DIR/.cache/npm"
"$NPM" --prefix web run build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/adn-report" .

"${SSH[@]}" "$REMOTE" "mkdir -p '$REMOTE_DIR/.staging' '$REMOTE_DIR/etc' '$REMOTE_DIR/data/csv' /data/log/go/adt-go/adn-report-ai; chmod 700 '$REMOTE_DIR/.staging' '$REMOTE_DIR/etc' '$REMOTE_DIR/data' '$REMOTE_DIR/data/csv' /data/log/go/adt-go/adn-report-ai"
"${SCP[@]}" "$BUILD_DIR/adn-report" "$REMOTE:$REMOTE_DIR/.staging/adn-report"
"${SCP[@]}" deploy/adn-report-ai.service deploy/activate.sh "$REMOTE:$REMOTE_DIR/.staging/"
if [[ "$REMOTE_CONFIG" == false || "$SYNC_CONFIG" == true ]]; then
    "${SCP[@]}" "$BUILD_DIR/config.yaml" "$REMOTE:$REMOTE_DIR/.staging/config.yaml"
fi
"${SSH[@]}" "$REMOTE" "bash '$REMOTE_DIR/.staging/activate.sh' '$STAGE' '$SYNC_CONFIG'"
