#!/usr/bin/env bash

set -uo pipefail

SERVICE_NAME="sublink"
INSTALL_DIR="/usr/local/bin/sublink"
BINARY="$INSTALL_DIR/sublink"
SERVICE_FILE="/etc/systemd/system/sublink.service"
REPOSITORY="hbsx/sublinkX"

if [ "$(id -u)" -ne 0 ]; then
    echo "请以 root 身份运行 sublink 菜单。" >&2
    exit 1
fi

get_latest_release() {
    curl -fsSL "https://api.github.com/repos/$REPOSITORY/releases/latest" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n 1
}

get_asset_name() {
    case "$(uname -m)" in
        x86_64|amd64) echo "sublink_amd64" ;;
        aarch64|arm64) echo "sublink_arm64" ;;
        *) return 1 ;;
    esac
}

update_sublink() {
    local latest_release current_version asset_name temporary_file

    latest_release=$(get_latest_release) || true
    if [ -z "$latest_release" ]; then
        echo "无法获取最新发行版标签。" >&2
        return 1
    fi

    current_version=$($BINARY --version 2>/dev/null || true)
    if [ "$current_version" = "$latest_release" ]; then
        echo "当前已经是最新版本: $current_version"
        return 0
    fi

    asset_name=$(get_asset_name) || {
        echo "不支持的机器类型: $(uname -m)" >&2
        return 1
    }

    temporary_file=$(mktemp)
    if ! curl -fL --retry 3 \
        "https://github.com/$REPOSITORY/releases/download/$latest_release/$asset_name" \
        -o "$temporary_file"; then
        rm -f "$temporary_file"
        echo "下载更新失败，原服务未受影响。" >&2
        return 1
    fi

    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    if ! install -m 0755 "$temporary_file" "$BINARY"; then
        rm -f "$temporary_file"
        systemctl start "$SERVICE_NAME" 2>/dev/null || true
        echo "安装更新失败，已尝试恢复服务。" >&2
        return 1
    fi
    rm -f "$temporary_file"

    if systemctl start "$SERVICE_NAME"; then
        echo "更新完成: ${current_version:-未知版本} -> $latest_release"
    else
        echo "更新已安装，但服务启动失败。请运行 systemctl status $SERVICE_NAME 查看原因。" >&2
        return 1
    fi
}

change_port() {
    local port

    read -r -p "请输入新的端口号: " port
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo "端口必须是 1 到 65535 之间的数字。" >&2
        return 1
    fi

    if [ ! -f "$SERVICE_FILE" ]; then
        echo "服务文件不存在: $SERVICE_FILE" >&2
        return 1
    fi

    sed -i "s|^ExecStart=.*|ExecStart=$BINARY run --port $port|" "$SERVICE_FILE"
    systemctl daemon-reload
    systemctl restart "$SERVICE_NAME"
    echo "端口已修改为 $port，服务已重启。"
}

reset_credentials() {
    local username password

    read -r -p "请输入新的账号: " username
    read -r -s -p "请输入新的密码: " password
    echo

    if [ -z "$username" ] || [ -z "$password" ]; then
        echo "账号和密码不能为空。" >&2
        return 1
    fi

    (cd "$INSTALL_DIR" && "$BINARY" setting --username "$username" --password "$password")
    systemctl restart "$SERVICE_NAME"
    echo "账号密码已重置，服务已重启。"
}

uninstall_sublink() {
    local delete_data

    systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
    rm -f "$SERVICE_FILE" "$BINARY" /usr/bin/sublink
    systemctl daemon-reload

    read -r -p "是否删除数据库、模板和日志？此操作不可恢复 (y/N): " delete_data
    if [[ "$delete_data" =~ ^[Yy]$ ]]; then
        rm -rf "$INSTALL_DIR/db" "$INSTALL_DIR/template" "$INSTALL_DIR/logs"
    fi

    rmdir "$INSTALL_DIR" 2>/dev/null || true
    echo "卸载完成。"
}

while true; do
    latest_release=$(get_latest_release 2>/dev/null || true)
    current_version=$($BINARY --version 2>/dev/null || echo "未安装")
    service_status=$(systemctl is-active "$SERVICE_NAME" 2>/dev/null || true)

    echo "最新版本: ${latest_release:-获取失败}"
    echo "当前版本: $current_version"
    echo "当前运行状态: ${service_status:-未知}"
    echo "1. 启动服务"
    echo "2. 停止服务"
    echo "3. 卸载"
    echo "4. 查看服务状态"
    echo "5. 查看运行目录"
    echo "6. 修改端口"
    echo "7. 更新"
    echo "8. 重置账号密码"
    echo "0. 退出"
    read -r -p "请选择一个选项: " option

    case "$option" in
        1)
            systemctl daemon-reload
            systemctl start "$SERVICE_NAME"
            ;;
        2)
            systemctl stop "$SERVICE_NAME"
            ;;
        3)
            uninstall_sublink
            exit 0
            ;;
        4)
            systemctl status "$SERVICE_NAME" --no-pager
            ;;
        5)
            echo "运行目录: $INSTALL_DIR"
            ls -la "$INSTALL_DIR"
            ;;
        6)
            change_port
            ;;
        7)
            update_sublink
            ;;
        8)
            reset_credentials
            ;;
        0)
            exit 0
            ;;
        *)
            echo "无效的选项，请重新选择。"
            ;;
    esac

    echo
done
