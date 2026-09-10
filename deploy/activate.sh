#!/usr/bin/env bash
set -euo pipefail

STAGE="${1:-false}"
SYNC_CONFIG="${2:-false}"
APP_DIR=/data/adn-report-ai
SERVICE=adn-report-ai.service
UNIT=/etc/systemd/system/adn-report-ai.service
CONFIG="$APP_DIR/etc/config.yaml"
BACKUP="$APP_DIR/backups/$(date +%Y%m%d-%H%M%S)-$$"
WAS_ACTIVE=false
CONFIG_CHANGED=false
UNIT_CHANGED=false
BINARY_CHANGED=false
if systemctl is-active --quiet "$SERVICE"; then WAS_ACTIVE=true; fi
mkdir -p "$BACKUP"
chmod 700 "$APP_DIR/backups" "$BACKUP"

rollback() {
    trap - ERR
    echo "部署失败，恢复本次替换的文件..." >&2
    if [[ "$BINARY_CHANGED" == true ]]; then
        systemctl stop "$SERVICE" || true
        if [[ -f "$BACKUP/adn-report" ]]; then cp -p "$BACKUP/adn-report" "$APP_DIR/adn-report"; fi
    fi
    if [[ "$CONFIG_CHANGED" == true && -f "$BACKUP/config.yaml" ]]; then cp -p "$BACKUP/config.yaml" "$CONFIG"; fi
    if [[ "$UNIT_CHANGED" == true && -f "$BACKUP/service" ]]; then cp -p "$BACKUP/service" "$UNIT"; fi
    systemctl daemon-reload
    if [[ "$WAS_ACTIVE" == true && "$BINARY_CHANGED" == true ]]; then systemctl start "$SERVICE" || true; fi
    echo "备份位置：$BACKUP" >&2
    exit 1
}
trap rollback ERR

if [[ ! -f "$CONFIG" || "$SYNC_CONFIG" == true ]]; then
    if [[ -f "$CONFIG" ]]; then cp -p "$CONFIG" "$BACKUP/config.yaml"; fi
    install -m 600 "$APP_DIR/.staging/config.yaml" "$CONFIG"
    CONFIG_CHANGED=true
else
    chmod 600 "$CONFIG"
    echo "保留远端私有配置。"
fi
chmod 755 "$APP_DIR/.staging/adn-report"
if [[ "$STAGE" == false ]]; then
    "$APP_DIR/.staging/adn-report" -f "$CONFIG" -check
    if [[ "$WAS_ACTIVE" == false && -n "$(ss -ltnH '( sport = :18080 )')" ]]; then
        echo "18080 已被其他服务占用，停止部署。" >&2
        false
    fi
fi

if [[ -f "$UNIT" ]]; then cp -p "$UNIT" "$BACKUP/service"; fi
UNIT_CHANGED=true
install -m 644 "$APP_DIR/.staging/adn-report-ai.service" "$UNIT"
systemctl daemon-reload

if [[ "$STAGE" == true ]]; then
    if [[ ! -f "$APP_DIR/adn-report" ]]; then install -m 755 "$APP_DIR/.staging/adn-report" "$APP_DIR/adn-report"; fi
    echo "应用文件已暂存；未启动或重启服务。"
    exit 0
fi

if [[ -f "$APP_DIR/adn-report" ]]; then cp -p "$APP_DIR/adn-report" "$BACKUP/adn-report"; fi
BINARY_CHANGED=true
install -m 755 "$APP_DIR/.staging/adn-report" "$APP_DIR/adn-report.new"
mv -f "$APP_DIR/adn-report.new" "$APP_DIR/adn-report"
systemctl restart "$SERVICE"
READY=false
for attempt in {1..30}; do
    if systemctl is-active --quiet "$SERVICE" && curl --connect-timeout 2 --max-time 3 -fsS -o /dev/null http://127.0.0.1:18080/api/auth/config; then
        READY=true
        break
    fi
    sleep 1
done
if [[ "$READY" != true ]]; then
    echo "服务未就绪，请检查 /data/log/go/adt-go/adn-report-ai/error.log" >&2
    false
fi
systemctl enable "$SERVICE"
echo "部署完成：172.16.3.34 上的应用 18080 端口已就绪。"
