#!/usr/bin/env bash
# Sourced by deployment scripts. Only changes PATH within the current process.
adn_node_supported() {
    "$1" -e 'const [major, minor] = process.versions.node.split(".").map(Number); process.exit(major > 22 || (major === 22 && minor >= 12) ? 0 : 1)' >/dev/null 2>&1
}

adn_setup_node() {
    local current candidate selected
    current="$(command -v node || true)"
    selected=""
    for candidate in "$current" /opt/homebrew/bin/node /usr/local/bin/node "${NVM_DIR:-$HOME/.nvm}"/versions/node/v*/bin/node; do
        if [[ -x "$candidate" ]] && adn_node_supported "$candidate"; then
            selected="$candidate"
            break
        fi
    done
    if [[ -z "$selected" ]]; then
        echo "未找到 Node >=22.12。请安装 Node 22 LTS 后重试；部署尚未执行。" >&2
        return 1
    fi
    export PATH="$(dirname -- "$selected"):$PATH"
    hash -r
    if ! command -v "${NPM:-npm}" >/dev/null 2>&1; then
        echo "已找到兼容的 Node，但未找到 npm，请检查 Node 安装。" >&2
        return 1
    fi
    printf '使用 Node %s（%s）\n' "$(node --version)" "$(command -v node)"
}
