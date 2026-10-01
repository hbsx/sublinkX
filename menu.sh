#!/usr/bin/env bash
set -uo pipefail
SERVICE_NAME="sublink"
INSTALL_DIR="/usr/local/bin/sublink"
BINARY="$INSTALL_DIR/sublink"
SERVICE_FILE="/etc/systemd/system/sublink.service"
MENU_FILE="/usr/bin/sublink"
RESTART_FILE="/etc/systemd/system/sublink.service.d/restart.conf"
REPOSITORY="hbsx/sublinkX"
if [ "$(id -u)" -ne 0 ]; then echo "请以 root 身份运行。" >&2; exit 1; fi
get_latest_release() {
    curl -fsSL --connect-timeout 10 --max-time 30 "https://api.github.com/repos/$REPOSITORY/releases/latest" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}
get_asset_name() {
    case "$(uname -m)" in
        x86_64|amd64) echo sublink_amd64 ;;
        aarch64|arm64) echo sublink_arm64 ;;
        *) return 1 ;;
    esac
}
download_asset() {
    local tag="$1" asset="$2" destination="$3" checksums expected
    checksums=$(mktemp "$INSTALL_DIR/.checksums.XXXXXX") || return 1
    if ! curl -fL --retry 3 --connect-timeout 10 --max-time 180 \
        "https://github.com/$REPOSITORY/releases/download/$tag/$asset" -o "$destination" ||
       ! curl -fsSL --retry 3 --connect-timeout 10 --max-time 30 \
        "https://github.com/$REPOSITORY/releases/download/$tag/SHA256SUMS" -o "$checksums"; then
        rm -f "$checksums"; return 1
    fi
    expected=$(awk -v asset="$asset" '$2 == asset {print $1}' "$checksums")
    rm -f "$checksums"
    if ! [[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]] ||
       ! printf '%s  %s\n' "$expected" "$destination" | sha256sum --check --status; then
        echo "发布包校验失败，已中止。" >&2; return 1
    fi
    chmod 0755 "$destination" && (cd "$INSTALL_DIR" && "$destination" --version >/dev/null)
}
restart_and_wait() {
    local deadline=$((SECONDS+30)) successes=0
    systemctl restart "$SERVICE_NAME" || return 1
    while [ "$SECONDS" -lt "$deadline" ]; do
        if systemctl is-active --quiet "$SERVICE_NAME" &&
           (cd "$INSTALL_DIR" && "$BINARY" healthcheck >/dev/null 2>&1); then
            successes=$((successes+1))
            [ "$successes" -ge 2 ] && return 0
        else successes=0; fi
        sleep 1
    done
    return 1
}
backup_state() {
    STATE_DIR=$(mktemp -d "$INSTALL_DIR/.rollback.XXXXXX") || return 1
    WAS_ACTIVE=0; WAS_ENABLED=0
    systemctl is-active --quiet "$SERVICE_NAME" && WAS_ACTIVE=1
    systemctl is-enabled --quiet "$SERVICE_NAME" && WAS_ENABLED=1
    if [ -f "$SERVICE_FILE" ]; then
        systemctl stop "$SERVICE_NAME" || { rm -rf "$STATE_DIR"; return 1; }
    fi
    local source name
    for name in binary menu service restart; do
        case "$name" in
            binary) source="$BINARY" ;; menu) source="$MENU_FILE" ;;
            service) source="$SERVICE_FILE" ;; restart) source="$RESTART_FILE" ;;
        esac
        if [ -f "$source" ]; then
            if ! cp -p "$source" "$STATE_DIR/$name"; then
                [ "$WAS_ACTIVE" -eq 1 ] && systemctl start "$SERVICE_NAME"
                rm -rf "$STATE_DIR"; return 1
            fi
        fi
    done
    local directories=()
    [ -d "$INSTALL_DIR/db" ] && directories+=(db)
    [ -d "$INSTALL_DIR/template" ] && directories+=(template)
    if [ "${#directories[@]}" -gt 0 ]; then
        if ! tar -czf "$STATE_DIR/data.tar.gz" -C "$INSTALL_DIR" "${directories[@]}"; then
            [ "$WAS_ACTIVE" -eq 1 ] && systemctl start "$SERVICE_NAME"
            rm -rf "$STATE_DIR"; return 1
        fi
    fi
    return 0
}
restore_state() {
    if ! systemctl stop "$SERVICE_NAME" 2>/dev/null; then
        echo "无法停止服务，未覆盖运行数据。备份保留于: $STATE_DIR" >&2
        return 1
    fi
    local target name failed=0
    for name in binary menu service restart; do
        case "$name" in
            binary) target="$BINARY" ;; menu) target="$MENU_FILE" ;;
            service) target="$SERVICE_FILE" ;; restart) target="$RESTART_FILE" ;;
        esac
        if [ -f "$STATE_DIR/$name" ]; then
            mkdir -p "$(dirname "$target")"
            cp -p "$STATE_DIR/$name" "$target.rollback" && mv -f "$target.rollback" "$target" || failed=1
        else rm -f "$target"; fi
    done
    if [ -f "$STATE_DIR/data.tar.gz" ]; then
        rm -rf -- "$INSTALL_DIR/db" "$INSTALL_DIR/template"
        tar -xzf "$STATE_DIR/data.tar.gz" -C "$INSTALL_DIR" || failed=1
    fi
    systemctl daemon-reload || failed=1
    if [ "$WAS_ENABLED" -eq 1 ]; then systemctl enable "$SERVICE_NAME" >/dev/null 2>&1;
    else systemctl disable "$SERVICE_NAME" >/dev/null 2>&1 || true; fi
    if [ "$WAS_ACTIVE" -eq 1 ]; then systemctl start "$SERVICE_NAME" || failed=1; fi
    if [ "$failed" -eq 0 ]; then
        rm -rf "$STATE_DIR"; echo "已恢复更新前的程序、配置和数据库。" >&2
    else echo "自动恢复未完全成功，备份保留于: $STATE_DIR" >&2; fi
    return "$failed"
}
install_candidate() {
    local candidate="$1" menu="${2:-}"
    backup_state || return 1
    if ! mv -f "$candidate" "$BINARY"; then restore_state; return 1; fi
    if [ -n "$menu" ]; then
        if ! install -m 0755 "$menu" "$MENU_FILE.new" || ! mv -f "$MENU_FILE.new" "$MENU_FILE"; then
            restore_state; return 1
        fi
        if [ ! -f "$SERVICE_FILE" ]; then
            if ! cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=Sublink Service
Wants=network-online.target
After=network-online.target
[Service]
Type=simple
ExecStart=$BINARY
WorkingDirectory=$INSTALL_DIR
Restart=on-failure
RestartSec=5s
[Install]
WantedBy=multi-user.target
EOF
            then restore_state; return 1; fi
        fi
        mkdir -p "$(dirname "$RESTART_FILE")"
        printf '[Service]\nRestart=on-failure\nRestartSec=5s\n' >"$RESTART_FILE" || { restore_state; return 1; }
    fi
    if systemctl daemon-reload && systemctl enable "$SERVICE_NAME" && restart_and_wait; then
        rm -rf "$STATE_DIR"; return 0
    fi
    echo "新版本启动验证失败，正在恢复。" >&2
    restore_state
    return 1
}
update_sublink() {
    local tag current asset candidate
    tag=$(get_latest_release) || true
    [ -n "$tag" ] || { echo "无法获取发行版。" >&2; return 1; }
    current=$(cd "$INSTALL_DIR" && "$BINARY" --version 2>/dev/null) || true
    [ "${current#v}" != "${tag#v}" ] || { echo "当前已是最新版: $current"; return 0; }
    asset=$(get_asset_name) || { echo "不支持的架构。" >&2; return 1; }
    candidate=$(mktemp "$INSTALL_DIR/.update.XXXXXX") || return 1
    if ! download_asset "$tag" "$asset" "$candidate"; then rm -f "$candidate"; return 1; fi
    if install_candidate "$candidate"; then echo "更新完成: $tag"; else rm -f "$candidate"; return 1; fi
}
change_port() {
    local port
    read -r -p "请输入新的端口号: " port || return 1
    [[ "$port" =~ ^[0-9]{1,5}$ ]] || { echo "端口无效。" >&2; return 1; }
    port=$((10#$port))
    if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then echo "端口须为 1～65535。" >&2; return 1; fi
    [ -f "$SERVICE_FILE" ] || { echo "服务文件不存在。" >&2; return 1; }
    backup_state || return 1
    if sed -i "s|^ExecStart=.*|ExecStart=$BINARY run --port $port|" "$SERVICE_FILE" &&
       systemctl daemon-reload && restart_and_wait; then
        rm -rf "$STATE_DIR"; echo "端口已修改为 $port，服务验证通过。"
    else echo "端口修改失败，正在恢复。" >&2; restore_state; return 1; fi
}
reset_credentials() {
    local username password
    read -r -p "请输入新的账号: " username || return 1
    read -r -s -p "请输入新的密码: " password || return 1
    echo
    if [ -z "$username" ] || [ "${#password}" -lt 6 ]; then echo "账号不能为空，密码至少为 6 位。" >&2; return 1; fi
    (cd "$INSTALL_DIR" && "$BINARY" setting --username "$username" --password "$password") || return 1
    restart_and_wait || return 1
    echo "账号密码已重置，旧登录已失效。"
}
uninstall_sublink() {
    local delete_data
    systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
    rm -f "$SERVICE_FILE" "$BINARY" "$MENU_FILE" "$RESTART_FILE"
    rmdir "$(dirname "$RESTART_FILE")" 2>/dev/null || true
    systemctl daemon-reload
    read -r -p "是否删除数据库、模板和日志？不可恢复 (y/N): " delete_data || return 1
    if [[ "$delete_data" =~ ^[Yy]$ ]]; then rm -rf "$INSTALL_DIR/db" "$INSTALL_DIR/template" "$INSTALL_DIR/logs"; fi
    rmdir "$INSTALL_DIR" 2>/dev/null || true
    echo "卸载完成。"
}
menu_main() {
    local latest current status option
    latest=$(get_latest_release 2>/dev/null || true)
    while true; do
        current=$(cd "$INSTALL_DIR" && "$BINARY" --version 2>/dev/null || echo "未安装")
        status=$(systemctl is-active "$SERVICE_NAME" 2>/dev/null || true)
        echo "最新版本: ${latest:-获取失败}  当前版本: $current  状态: ${status:-未知}"
        echo "1. 启动  2. 停止  3. 卸载  4. 状态  5. 目录  6. 修改端口  7. 更新  8. 重置账号  0. 退出"
        read -r -p "请选择: " option || return 0
        case "$option" in
            1) restart_and_wait ;; 2) systemctl stop "$SERVICE_NAME" ;;
            3) uninstall_sublink; return ;; 4) systemctl status "$SERVICE_NAME" --no-pager ;;
            5) ls -la "$INSTALL_DIR" ;; 6) change_port ;; 7) update_sublink ;;
            8) reset_credentials ;; 0) return ;; *) echo "无效选项。" ;;
        esac
    done
}
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then menu_main; fi
