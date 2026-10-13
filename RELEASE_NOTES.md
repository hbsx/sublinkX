## 2.1.5 · 2026-10-13

### 修复 TUIC 订阅

- 正确解析从 v2rayN 导出的 `UUID%3A密码`，分别写入 Mihomo 节点的 `uuid` 和 `password`。
- 将 TUIC 链接中的 `allow_insecure=1` 作为单节点 `skip-cert-verify: true` 输出；支持小写的拥塞控制和 UDP 转发参数，并使用 Mihomo 对应字段。
- 增加 TUIC 解析和 Clash、Surge 输出的回归测试。

### 升级与验证

- Docker 镜像：`ghcr.io/hbsx/sublinkx:2.1.5`。升级前备份数据，重建容器时保留原有 `db`、`template`、`logs` 挂载和账号，再刷新 Mihomo 订阅。仅重启 `2.1.4` 容器不会更新程序。
- Docker 构建覆盖 Linux AMD64、ARMv7、ARM64；Release 提供 Linux AMD64/ARM64、Windows AMD64 和 `SHA256SUMS`。
- `go test ./...` 通过；GitHub Actions 的完整测试和多架构镜像构建通过。
