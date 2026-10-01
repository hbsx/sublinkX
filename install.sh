#!/usr/bin/env bash
set -euo pipefail
if [ "$(id -u)" -ne 0 ]; then echo "请以 root 身份运行。" >&2; exit 1; fi
INSTALL_DIR="/usr/local/bin/sublink"
REPOSITORY="hbsx/sublinkX"
for required in curl systemctl tar sha256sum; do
    command -v "$required" >/dev/null || { echo "缺少必要命令: $required" >&2; exit 1; }
done
mkdir -p "$INSTALL_DIR/db" "$INSTALL_DIR/template" "$INSTALL_DIR/logs"
temporary_dir=$(mktemp -d "$INSTALL_DIR/.install.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
curl -fsSL --retry 3 --connect-timeout 10 --max-time 30 \
    "https://raw.githubusercontent.com/$REPOSITORY/main/menu.sh" -o "$temporary_dir/menu.sh"
bash -n "$temporary_dir/menu.sh"
source "$temporary_dir/menu.sh"
tag=$(get_latest_release)
[ -n "$tag" ] || { echo "无法获取发行版。" >&2; exit 1; }
asset=$(get_asset_name) || { echo "不支持的架构。" >&2; exit 1; }
download_asset "$tag" "$asset" "$temporary_dir/sublink"
install_candidate "$temporary_dir/sublink" "$temporary_dir/menu.sh"
echo "安装完成: $tag。服务启动验证通过并已设置开机启动。"
echo "默认账号 admin，默认密码 123456，默认端口 8000。已有安装保留原账号和端口。"
echo "输入 sublink 打开菜单。升级后请重新登录，并从后台重新复制订阅链接。"
