#!/usr/bin/env bash

set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "该脚本必须以 root 身份运行。" >&2
    exit 1
fi

INSTALL_DIR="/usr/local/bin/sublink"
REPOSITORY="hbsx/sublinkX"

for command_name in curl systemctl; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
        echo "缺少必要命令: $command_name" >&2
        exit 1
    fi
done

mkdir -p "$INSTALL_DIR/db" "$INSTALL_DIR/logs" "$INSTALL_DIR/template"

latest_release=$(curl -fsSL "https://api.github.com/repos/$REPOSITORY/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1)

if [ -z "$latest_release" ]; then
    echo "无法获取最新发行版标签。" >&2
    exit 1
fi

echo "最新版本: $latest_release"

case "$(uname -m)" in
    x86_64|amd64)
        file_name="sublink_amd64"
        ;;
    aarch64|arm64)
        file_name="sublink_arm64"
        ;;
    *)
        echo "不支持的机器类型: $(uname -m)" >&2
        exit 1
        ;;
esac

temporary_file=$(mktemp)
trap 'rm -f "$temporary_file"' EXIT

curl -fL --retry 3 \
    "https://github.com/$REPOSITORY/releases/download/$latest_release/$file_name" \
    -o "$temporary_file"
install -m 0755 "$temporary_file" "$INSTALL_DIR/sublink"

cat >/etc/systemd/system/sublink.service <<EOF
[Unit]
Description=Sublink Service
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/sublink
WorkingDirectory=$INSTALL_DIR
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable sublink
systemctl restart sublink

curl -fL --retry 3 \
    -H "Cache-Control: no-cache" \
    -H "Pragma: no-cache" \
    "https://raw.githubusercontent.com/$REPOSITORY/main/menu.sh" \
    -o /usr/bin/sublink
chmod 0755 /usr/bin/sublink

echo "服务已启动并已设置为开机启动。"
echo "默认账号 admin，默认密码 123456，默认端口 8000。"
echo "安装完成后输入 sublink 可以呼出菜单。"
