#!/usr/bin/env bash
# 在 Linux 线上仓库目录执行；使用 scripts/build-linux.sh 生成的预编译产物。
# 用法：./deploy.sh；非默认监听端口：PORT=18081 ./deploy.sh
set -euo pipefail

APP_NAME=adn-report-ai
REPO_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ARTIFACT="$REPO_DIR/dist-linux/adn-report"
CONFIG="$REPO_DIR/etc/config.yaml"
BINARY="$REPO_DIR/bin/adn-report"
UNIT="/etc/systemd/system/$APP_NAME.service"
PORT="${PORT:-18080}"

[[ "$(uname -s)" == Linux ]] || { echo '请在 Linux 服务器执行此脚本。' >&2; exit 1; }
[[ "$PORT" =~ ^[0-9]+$ ]] && (( PORT > 0 && PORT < 65536 )) || { echo 'PORT 必须为有效端口。' >&2; exit 1; }
# 避免路径在 systemd 指令中被解释为转义或 specifier。
case "$REPO_DIR" in *'%'*|*'"'*|*'\'*|*$'\n'*) echo '仓库路径包含 systemd 不支持的特殊字符。' >&2; exit 1 ;; esac
SUDO=()
if [[ "$EUID" -ne 0 ]]; then SUDO=(sudo); fi
for command in git curl systemctl install; do command -v "$command" >/dev/null; done
cd "$REPO_DIR"

echo '==> [1/5] 拉取最新代码'
git pull --ff-only

echo '==> [2/5] 校验产物与私有配置'
if [[ ! -f "$ARTIFACT" ]]; then
    echo '缺少 dist-linux/adn-report。请先在开发机执行 bash scripts/build-linux.sh，并提交、推送产物及代码。' >&2
    exit 1
fi
if [[ ! -f "$CONFIG" ]]; then
    echo '缺少 etc/config.yaml。请根据 etc/config.example.yaml 创建线上私有配置，填写数据库、登录和实际 BaseURL；不会覆盖已有配置。' >&2
    exit 1
fi
chmod 755 "$ARTIFACT"
"${SUDO[@]}" chmod 600 "$CONFIG"
# 与服务使用同一工作目录；检查配置和数据库，不初始化表、不启动 HTTP。
"${SUDO[@]}" "$ARTIFACT" -f "$CONFIG" -check

BACKUP="$REPO_DIR/.cache/deploy-backups/$(date +%Y%m%d-%H%M%S)-$$"
mkdir -p "$BACKUP" "$REPO_DIR/bin"
chmod 700 "$BACKUP"
WAS_ACTIVE=false
HAD_BINARY=false
HAD_UNIT=false
if systemctl is-active --quiet "$APP_NAME"; then WAS_ACTIVE=true; fi
if [[ -f "$BINARY" ]]; then cp -p "$BINARY" "$BACKUP/adn-report"; HAD_BINARY=true; fi
if [[ -f "$UNIT" ]]; then "${SUDO[@]}" cp -p "$UNIT" "$BACKUP/service"; HAD_UNIT=true; fi
rollback() {
    trap - ERR
    echo "部署失败，恢复旧程序及服务配置；备份：$BACKUP" >&2
    "${SUDO[@]}" systemctl stop "$APP_NAME" || true
    if [[ "$HAD_BINARY" == true ]]; then
        cp -p "$BACKUP/adn-report" "$BINARY.new"
        mv -f "$BINARY.new" "$BINARY"
    fi
    if [[ "$HAD_UNIT" == true ]]; then
        "${SUDO[@]}" cp -p "$BACKUP/service" "$UNIT"
    else
        "${SUDO[@]}" rm -f "$UNIT"
    fi
    "${SUDO[@]}" systemctl daemon-reload
    if [[ "$WAS_ACTIVE" == true ]]; then "${SUDO[@]}" systemctl start "$APP_NAME" || true; fi
    exit 1
}
trap rollback ERR

echo '==> [3/5] 安装程序及 systemd 服务'
# 运行文件独立于 git 产物，原子替换，保留失败回退所需的旧程序。
install -m 755 "$ARTIFACT" "$BINARY.new"
mv -f "$BINARY.new" "$BINARY"
cat > "$BACKUP/new.service" <<UNITFILE
[Unit]
Description=ADN Settlement Report Platform
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory="$REPO_DIR"
ExecStart="$BINARY" -f "$CONFIG"
Restart=on-failure
RestartSec=5
TimeoutStopSec=20
UMask=0077
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNITFILE
"${SUDO[@]}" install -m 644 "$BACKUP/new.service" "$UNIT"
"${SUDO[@]}" systemctl daemon-reload

echo '==> [4/5] 重启服务'
"${SUDO[@]}" systemctl restart "$APP_NAME"

echo '==> [5/5] 检查服务就绪'
READY=false
for attempt in {1..30}; do
    if systemctl is-active --quiet "$APP_NAME" && curl --connect-timeout 2 --max-time 3 -fsS -o /dev/null "http://127.0.0.1:$PORT/api/auth/config"; then
        READY=true
        break
    fi
    sleep 1
done
if [[ "$READY" != true ]]; then
    echo "服务未就绪，请检查配置监听端口是否为 $PORT；日志：sudo journalctl -u $APP_NAME -n 50 --no-pager" >&2
    false
fi
"${SUDO[@]}" systemctl enable "$APP_NAME"
trap - ERR
echo "部署成功：$APP_NAME 已就绪；对外访问地址以 etc/config.yaml 的 BaseURL 为准。"
