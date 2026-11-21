# 登录 IP 记录与 Telegram 提醒

后台右上角账号菜单 → 账号设置（`/#/system/user/set`），可查看「登录记录」和配置「Telegram 登录提醒」。

## 记录范围与存储

- 记录每次账号密码登录的时间、账号、来源 IP、IP 识别方式、浏览器和成功/失败结果。
- **最多保留最近 20 条，按时间倒序显示；新记录覆盖最旧记录。**成功与失败共用这 20 条额度。
- 记录只在进程内存中，不创建数据库表、不写登录历史文件，容器或服务重启后清空。
- 已登录会话的普通页面访问、订阅拉取和端口扫描不算一次登录，不会生成登录记录。
- 页面需要现有后台登录凭据才能查询或修改设置；接口返回 `Cache-Control: no-store`。
- 历史数据不包含密码、JWT 或机器人 Token。浏览器和账号字段为有限长度的文本，不按 HTML 渲染。

## Telegram 配置

在账号设置填写机器人 Bot Token、接收 Chat ID，启用并保存。可复用现有机器人：这里只调用 `sendMessage`，不注册命令、不接收更新、不影响原机器人的指令。

- 每次成功登录发送一条提醒，包含时间、账号、IP、识别方式和设备信息。
- 5 分钟内累计 5 次失败时发送合并提醒，失败提醒最多每 5 分钟一条，多个 IP 共用这个限制。
- 发送使用有界内存队列，网络超时 8 秒，不阻塞登录。队列满或 Telegram 不可达时可能丢失提醒，不在磁盘排队，也不自动重试；后台显示发送成功、失败、跳过数量。
- 点击「测试已保存配置」会异步发送测试提醒；排队成功不代表 Telegram 已送达，可点击「刷新」查看发送结果。
- Bot Token 不在查询接口中返回，编辑时留空保留原值；关闭提醒后保留配置，方便再次启用。
- **只有主动保存设置时**才将提醒配置写入 `db/login-security.json`（Linux 权限 `0600`），这是重启后继续提醒所需的少量持久配置。配置文件含 Token，不要上传 GitHub。

也可通过环境变量设置初始配置：

```dotenv
SUBLINK_LOGIN_TG_ENABLED=1
SUBLINK_LOGIN_TG_BOT_TOKEN=你的机器人Token
SUBLINK_LOGIN_TG_CHAT_ID=你的ChatID
SUBLINK_LOGIN_TG_PROXY_URL=http://host.docker.internal:7890
```

保存过的 `db/login-security.json` 优先于环境变量。代理只用于 Telegram，留空直连。Docker 中 `127.0.0.1` 指容器自身；Compose 已加入 `host.docker.internal` 映射，宿主机代理还须允许来自容器的连接。代理地址不接受账号密码，避免配置查询回显代理凭据。

## Cloudflare Tunnel 与真实 IP

默认忽略所有转发头，记录直接连接者的 IP，防止互联网访客自行伪造来源。通过可信反向代理访问时，显式设置：

```dotenv
# 仅示例：必须先确认实际连接者地址，不要直接照抄。
SUBLINK_LOGIN_TRUSTED_PROXIES=172.17.0.1
SUBLINK_LOGIN_IP_HEADER=CF-Connecting-IP
```

`SUBLINK_LOGIN_TRUSTED_PROXIES` 是**与 SublinkX 直接建立连接的代理**地址或精确网段，可用逗号分隔。只有连接来自这些地址，才接受指定来源头；Cloudflare Tunnel 使用 `CF-Connecting-IP`。普通反代可以改为 `X-Forwarded-For`，从右向左选择首个不可信代理的地址，防止左侧追加伪造 IP。无效/重复头退回直接连接地址。

不要信任所有地址或整个家庭内网。代理必须覆盖客户端传入的来源头。Docker 宿主机/容器 NAT 可能让不同入口显示相同代理地址，部署时应确认拓扑；需要严格隔离时，将 Tunnel 入口放在专用容器网络、限制后端入口，并只信任该连接器地址。IP 记录是来源线索，不能代替登录验证，也无法还原访客自身使用的 VPN/代理之前的地址。

## 减少日志写入

新增登录历史与通知队列始终在内存中。应用普通日志默认只输出控制台，不再默认创建每日文件；可设置：

```dotenv
SUBLINK_LOG_MODE=off
```

`off` 关闭应用日志，`console` 只输出控制台，`file` 显式恢复原有 `logs` 文件。仓库 Compose 默认 `off`，同时使用 Docker `logging.driver: none`，避免 Docker 再写一份日志。自定义 Docker 部署若要避免容器标准输出落盘，也应使用 `--log-driver none`。

账号、节点、订阅和 Telegram 设置仍需持久保存；本功能不会将现有业务数据库改为内存数据库。

## 构建与升级

Dockerfile 会构建 Vue 前端，再编译 Go，将新的账号设置页面嵌入镜像。Release 的 `build.sh` 同样先构建前端，需要 Go、Node 和 pnpm 9。

提交代码或重启旧容器不会自动更新正在运行的程序；需拉取包含本功能的新镜像并重建容器，保留原有 `db` 和 `template` 挂载。配置提醒后，可分别测试一次正确和错误密码登录，确认后台记录与 Telegram 提醒。
