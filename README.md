<div align="center">
<img src="webs/src/assets/logo.png" width="150px" height="150px" />
</div>

<div align="center">
    <img src="https://img.shields.io/badge/Vue-5.0.8-brightgreen.svg"/>
    <img src="https://img.shields.io/badge/Go-1.22.0-green.svg"/>
    <img src="https://img.shields.io/badge/Element Plus-2.6.1-blue.svg"/>
    <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
    <a href="https://t.me/+u6gLWF0yP5NiZWQ1" target="_blank">
        <img src="https://img.shields.io/badge/TG-交流群-orange.svg"/>
    </a>
    <div align="center"> 中文 | <a href="README.en-US.md">English</div>
</div>

## [项目简介]

项目基于sublink项目二次开发：https://github.com/jaaksii/sublink

前端基于：https://github.com/youlaitech/vue3-element-admin

后端采用go+gin+gorm

默认账号admin 密码123456  自行修改

因为重写目前还有很多布局结构以及功能稍少

## [项目特色]

* 订阅管理、节点分组、访问记录与模板配置集中在同一套 Web 管理界面。
* 同时提供 Linux 二进制安装、升级菜单与 Docker 构建方式；数据目录可独立挂载、便于备份迁移。
* 输出 v2ray Base64 通用订阅、Clash YAML 与 Surge 配置，按客户端自动过滤不兼容节点。
* 最新分支已移除登录验证码；首次使用后请立即修改默认管理员密码。

### 支持的节点协议

| 协议 | 订阅解析 | Clash 输出 | Surge 输出 |
| --- | --- | --- | --- |
| Shadowsocks (SS) / ShadowsocksR (SSR) | 支持 | 支持 | SS 支持 |
| VMess | 支持 | 支持 | 支持 |
| VLESS（含 XHTTP） | 支持 | 支持 | 客户端不支持时自动过滤 |
| Trojan | 支持 | 支持 | 支持 |
| Hysteria / Hysteria2 | 支持 | 支持 | Hysteria2 支持 |
| TUIC | 支持 | 支持 | 支持 |
| AnyTLS | 支持 | 支持 | 客户端不支持时自动过滤 |
| SOCKS5、HTTP、HTTPS 代理链接 | 支持 | 支持 | 客户端不支持时自动过滤 |

协议是否可用仍取决于订阅客户端自身版本；生成时会优先保留该客户端能够识别的节点，避免产生无效配置。

### 节点导入与导出

节点管理页面新增「导入节点」「导出节点」。新机器安装后，可从旧机器导出 JSON，在新机器预览并确认导入，再按新地址创建订阅。

- JSON 备份保留节点名称、原始链接、链接类型和分组；不包含数据库 ID、订阅配置、模板或系统设置。
- TXT 每行一个节点链接，导入时自动识别名称；不保留自定义名称、分组和链接类型，HTTP/HTTPS 代理建议使用 JSON 迁移。
- 可导出全部或选中节点；单次上限 5000 条记录、5 MB，超限请分批操作。
- 导入只追加，链接及链接类型相同的记录跳过（含文件内重复），原有名称、分组不变。
- 预览不写入数据。存在无效记录时不能确认；保存使用事务，失败则回滚本次导入。
- 文件含节点凭据和订阅令牌，请妥善保管。

## [项目预览]

![1712594176714](webs/src/assets/1.png)
![1712594176714](webs/src/assets/2.png)

## 更新说明

### [2.1.4](https://github.com/hbsx/sublinkX/releases/tag/2.1.4) · 2026-10-03

新增节点 JSON/TXT 导入导出。JSON 保留名称、链接、链接类型和分组；支持全部或选中导出、导入预览、重复跳过及失败回滚。仅迁移节点，不迁移订阅配置。详见上方「节点导入与导出」。

**升级：**备份数据后，用 `ghcr.io/hbsx/sublinkx:2.1.4` 重建容器，沿用原数据挂载和账号，强制刷新网页。从 2.1.3 升级保留原订阅配置。

### [2.1.3](https://github.com/hbsx/sublinkX/releases/tag/2.1.3) · 2026-10-02

- 修复错误 ID 误删节点，节点编辑与分组更新在同一事务中完成。
- 节点选择、订阅编辑、节点排序和访问日志使用固定 ID；旧名称排序自动迁移，保留现有关联及 2.1.2 的订阅 token。
- 修复节点重命名、同名、名称含逗号时的丢失和错位；Clash/Surge 输出代理名称冲突时自动添加后缀。
- 修复 VMess、SS、SSR 的异常输入解析，正确保留 SSR 密码、SS/SSR IPv6、VMess SNI 与 ALPN，并拒绝越界端口。
- 异常日志不再输出请求查询参数、认证头或 panic 值，补齐之前遗漏的异常路径。
- 节点 URL 中的逗号参数保留；批量输入建议用换行。远程 HTML、错误文本和无效本地节点返回错误，不再生成假成功订阅。
- Surge 自动更新地址识别反向代理的 HTTPS 协议；反向代理应覆盖客户端传入的 `X-Forwarded-Proto`。
- 修复连续编辑订阅时证书选项残留、删除全部节点后页面仍保留旧行的问题。
- 正在使用的模板禁止改名或删除；请先在订阅中更换模板，再进行操作。模板内容仍可原名编辑。

**升级：**先备份数据，拉取 `ghcr.io/hbsx/sublinkx:2.1.3` 并重建容器，继续挂载原目录。从 2.1.2 升级保留订阅链接；从更早版本升级仍需重新复制订阅链接。网页有缓存时强制刷新。安装命令仍使用本 fork。

完整说明和验证范围见 [RELEASE_NOTES.md](RELEASE_NOTES.md)。这些修复针对已确认的场景，不代表全部功能不存在其它缺陷。

### [2.1.2](https://github.com/hbsx/sublinkX/releases/tag/2.1.2) · 2026-10-02

本版本包含订阅、安全、模板和安装更新修复，并将当前分支的修复打包到 Docker 镜像与 Release。完整说明见 [RELEASE_NOTES.md](RELEASE_NOTES.md)。

#### 订阅与账号安全

- 修复多个订阅同时请求时可能读取其它订阅内容的问题，每个请求独立读取对应订阅。
- 使用持久保存的随机订阅凭据，替代按订阅名称生成的 MD5 链接；更改订阅名称和正常重启不会改变新凭据。
- 修改密码、重置账号或退出登录后撤销旧登录凭据；同时发生的旧密码登录也不能获得有效的新登录。
- 密码改为 bcrypt 哈希保存，旧数据库中的明文密码自动迁移，原账号和密码仍可用于重新登录。
- 移除密码、节点链接和订阅凭据等敏感日志输出；历史日志不会自动删除。

#### 代理识别与远程请求

- 修复 HTTP/HTTPS 代理链接被误当作远程订阅下载的问题，节点页面新增“自动识别 / 代理节点 / 远程订阅”选项。
- 自动识别将带账号密码、名称片段或仅带端口的 HTTP/HTTPS 链接视为代理。无认证、无名称且使用默认端口的代理请选择“代理节点”；有歧义的订阅请选择“远程订阅”。
- 远程订阅、模板和 IP 信息请求增加超时、响应状态检查和内容大小限制，避免长时间等待或无限读取。

#### 模板与命令行

- 修复 Surge 模板重复插入段落、破坏规则内容的问题。
- Clash 模板格式错误时返回错误，避免不安全的类型转换导致请求崩溃；遇到 relay 策略组后继续处理后续策略组。
- 端口统一按十进制解析，`08000` 表示 `8000`；无效端口会被拒绝。
- `--version` 和 `healthcheck` 不再创建配置、数据库或模板目录；健康检查同时核对服务版本。

#### Docker、安装与发布

- Docker 镜像 `ghcr.io/hbsx/sublinkx:2.1.2` 和 `latest` 支持 AMD64、ARMv7、ARM64。
- Release 提供 Linux AMD64/ARM64、Windows AMD64 程序及 `SHA256SUMS`，安装脚本使用本 fork 的下载地址。
- Linux 安装与菜单更新校验下载文件，停止服务后备份程序、数据库、配置和模板；启动验证失败时自动恢复。修改端口失败也会恢复原状态。
- 保留当前分支的 AnyTLS、SOCKS5、HTTP/HTTPS、VLESS XHTTP 支持和免验证码登录。

#### 升级注意事项

1. 先备份现有数据目录。数据库自动迁移，保留节点、订阅名称和模板，无需删除数据库。
2. **升级后需要重新登录，并从后台重新复制订阅链接。旧的 MD5 链接不再有效，请更新客户端中保存的链接。**
3. Docker 部署需要拉取新镜像并重建容器，单独重启原容器不会升级；继续挂载原有 `db`、`template`、`logs` 目录。Docker 升级不包含 Linux 安装脚本的自动回退功能。
4. Linux 二进制部署建议重新运行下方一键安装命令，以同时更新程序和管理菜单；已有账号、数据和端口会保留。

已通过 Go 自动测试与并发检查、前端构建、ARM64 容器启动/重启与旧数据库迁移，以及安装失败回退、端口恢复和下载校验测试。

### 2.1（历史版本）

- 增加节点分组，调整节点管理页面并修复订阅解析问题。
- 升级到 2.1.2 时，请按上述升级注意事项备份并保留原数据库。

## [安装说明]

当前分支的最新版本包含 AnyTLS、SOCKS5、HTTP/HTTPS、VLESS XHTTP 支持，并已移除登录验证码。

### linux方式：
```
curl -fsSL -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/hbsx/sublinkX/main/install.sh | sudo bash
```

```sublink``` 呼出菜单

然后输入安装脚本即可

### Docker 方式

#### Docker Compose（推荐）

```bash
git clone https://github.com/hbsx/sublinkX.git
cd sublinkX
docker compose up -d
```

Compose 会拉取本仓库自动构建的 AMD64、ARMv7 或 ARM64 镜像。运行数据保存在仓库目录下的 `data/db`、`data/template` 和 `data/logs` 中。配置已启用 `restart: unless-stopped`，Docker 服务或服务器重启后会自动恢复容器。

如需修改宿主机端口、固定镜像版本或使用已有数据，先复制 `.env.example` 为 `.env`。例如将 `SUBLINK_PORT=8080`，即可访问宿主机的 8080 端口，容器内的 `db/config.yaml` 仍保持 `port: 8000`。

从旧版 Compose 或 Docker run 迁移时，先用 `docker inspect sublinkx --format '{{json .Mounts}}'` 确认旧数据位置并备份。在 `.env` 中将 `SUBLINK_DATA_DIR` 设置为原来的数据父目录（包含 `db`、`template`、`logs`），然后停止并删除旧容器、启动 Compose。不要在未确认数据目录前直接使用默认的 `./data`，否则会创建一个空数据库。

如果曾主动执行 `docker stop` 或 `docker compose stop`，`unless-stopped` 会保留停止状态，需用 `docker start sublinkx` 或 `docker compose up -d` 恢复。健康检查显示 `unhealthy` 也不会自动触发 Docker 重启；请查看日志排查。

查看状态和日志：

```bash
docker compose ps
docker compose logs --tail=100 sublinkx
```

更新 Compose 部署：

```bash
cd sublinkX
git pull --ff-only
docker compose pull
docker compose up -d
```

#### Docker run

##### 1. 准备环境

服务器需要提前安装 Docker，并确保 `8000` 端口未被其他程序占用：

```bash
docker --version
```

##### 2. 创建数据目录

选择一个用于长期保存数据的位置，例如：

```bash
mkdir -p ~/sublinkx/db ~/sublinkx/template ~/sublinkx/logs
cd ~/sublinkx
```

三个目录的用途：

- `db`：保存数据库 `sublink.db` 和程序配置 `config.yaml`，必须备份。
- `template`：保存 Clash、Surge 等订阅模板，建议备份。
- `logs`：保存运行日志，可按需备份或清理。

不需要手工创建上述文件。容器首次启动后，程序会自动生成数据库、配置文件和默认模板。

##### 3. 拉取镜像

拉取与服务器架构匹配的最新版镜像：

```bash
docker pull ghcr.io/hbsx/sublinkx:latest
```

##### 4. 创建并启动容器

在 `~/sublinkx` 目录中执行：

```bash
docker run -d \
  --name sublinkx \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/db:/app/db" \
  -v "$PWD/template:/app/template" \
  -v "$PWD/logs:/app/logs" \
  ghcr.io/hbsx/sublinkx:latest
```

启动后检查容器状态和日志：

```bash
docker ps --filter name=sublinkx
docker logs --tail=100 sublinkx
```

如果容器是按旧版说明创建的，可以在不删除容器和数据的情况下补上自动重启策略：

```bash
docker update --restart unless-stopped sublinkx
```

浏览器访问：`http://服务器IP:8000`

默认账号：`admin`  
默认密码：`123456`

首次登录后请立即修改默认密码。

##### 5. 更新版本

数据已保存在宿主机的 `db`、`template` 和 `logs` 目录中，删除并重建容器不会丢失这些数据：

```bash
cd ~/sublinkx
docker pull ghcr.io/hbsx/sublinkx:latest
docker rm -f sublinkx
docker run -d \
  --name sublinkx \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/db:/app/db" \
  -v "$PWD/template:/app/template" \
  -v "$PWD/logs:/app/logs" \
  ghcr.io/hbsx/sublinkx:latest
```

##### 6. 备份数据

建议在更新前备份数据库和模板：

```bash
cd ~/sublinkx
docker stop sublinkx
tar -czf "sublinkx-backup-$(date +%F).tar.gz" db template
docker start sublinkx
```

##### 7. 可选：清理无用镜像

更新完成后，可以清理不再使用的旧镜像层：

```bash
docker image prune -f
```

该命令不会删除正在使用的镜像、容器或挂载的 `db`、`template` 和 `logs` 数据。
To support the development of my project, I plan to apply for a free VPS offered by ZMTO. My project currently involves Docker image support for multiple architectures (arm64 and amd64), as well as automation for building and pushing. Therefore, I am requesting a 4-core, 8GB RAM Ubuntu VPS with root access.

Thank you to the ZMTO team for your support. I look forward to leveraging this VPS to optimize my project's performance and development efficiency. If you have any questions or suggestions regarding my project, feel free to open an issue, and I will do my best to improve and optimize it.

Thank you for your attention and support!

Feel free to adjust any details as needed!

## Stargazers over time
[![Stargazers over time](https://starchart.cc/gooaclok819/sublinkX.svg?variant=adaptive)](https://starchart.cc/gooaclok819/sublinkX)
