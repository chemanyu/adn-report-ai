#!/usr/bin/env bash
# 在开发机构建包含 Vue 页面的 Linux 二进制，服务器无需 Node/Go。
set -euo pipefail
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source "$ROOT_DIR/scripts/node_env.sh"
adn_setup_node
export GOCACHE="$ROOT_DIR/.cache/go-build"
"${NPM:-npm}" --prefix web ci --cache "$ROOT_DIR/.cache/npm"
"${NPM:-npm}" --prefix web run build
mkdir -p dist-linux
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH:-amd64}" go build -o dist-linux/adn-report .
echo '已生成 dist-linux/adn-report（包含 Vue 页面，不含私有配置）。'
