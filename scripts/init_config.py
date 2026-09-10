"""Create a private local config once, without overwriting existing settings."""
from pathlib import Path
import os
import secrets

root = Path(__file__).resolve().parent.parent
target = root / "etc/config.yaml"
if target.exists():
    print("已存在 etc/config.yaml，保留当前配置。")
else:
    text = (root / "etc/config.example.yaml").read_text()
    text = text.replace('AdminPassword: ""', f'AdminPassword: "{secrets.token_urlsafe(18)}"')
    fd = os.open(target, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    with os.fdopen(fd, "w") as handle:
        handle.write(text)
    print("已创建 etc/config.yaml。管理员账号：admin；随机密码见 Auth.AdminPassword。请填写 PostgreSQL.DSN 或设置 DATABASE_URL 后启动。")
