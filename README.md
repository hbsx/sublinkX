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

## [项目预览]

![1712594176714](webs/src/assets/1.png)
![1712594176714](webs/src/assets/2.png)

## [2.1更新说明]

#### 后端更新

1. 修复底层代码
2. 修复各种奇葩bug
3. 建议卸载数据库(记得备份数据) 新数据库结构有些不一样可能会导致一些bug

#### 前端更新

1. 完善node页面




## [安装说明]

当前分支的最新版本包含 AnyTLS、SOCKS5、HTTP/HTTPS、VLESS XHTTP 支持，并已移除登录验证码。

### linux方式：
```
curl -s -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/hbsx/sublinkX/main/install.sh | sudo bash
```

```sublink``` 呼出菜单

然后输入安装脚本即可

### Docker 方式

#### 1. 准备环境

服务器需要提前安装 Docker，并确保 `8000` 端口未被其他程序占用：

```bash
docker --version
```

#### 2. 创建数据目录

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

#### 3. 构建镜像

直接从本仓库最新版源码构建：

```bash
docker build -t hbsx/sublinkx:latest https://github.com/hbsx/sublinkX.git
```

#### 4. 创建并启动容器

在 `~/sublinkx` 目录中执行：

```bash
docker run -d \
  --name sublinkx \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/db:/app/db" \
  -v "$PWD/template:/app/template" \
  -v "$PWD/logs:/app/logs" \
  hbsx/sublinkx:latest
```

启动后检查容器状态和日志：

```bash
docker ps --filter name=sublinkx
docker logs --tail=100 sublinkx
```

浏览器访问：`http://服务器IP:8000`

默认账号：`admin`  
默认密码：`123456`

首次登录后请立即修改默认密码。

#### 5. 更新版本

数据已保存在宿主机的 `db`、`template` 和 `logs` 目录中，删除并重建容器不会丢失这些数据：

```bash
cd ~/sublinkx
docker rm -f sublinkx
docker build -t hbsx/sublinkx:latest https://github.com/hbsx/sublinkX.git
docker run -d \
  --name sublinkx \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/db:/app/db" \
  -v "$PWD/template:/app/template" \
  -v "$PWD/logs:/app/logs" \
  hbsx/sublinkx:latest
```

#### 6. 备份数据

建议在更新前备份数据库和模板：

```bash
cd ~/sublinkx
tar -czf "sublinkx-backup-$(date +%F).tar.gz" db template
```

#### 7. 可选：清理构建缓存

如果构建成功后近期不再重新安装或更新，可以手动清理未使用的 Docker 构建缓存：

```bash
docker builder prune -f
```

该命令不会删除已经构建的镜像、正在运行的容器，也不会删除挂载的 `db`、`template` 和 `logs` 数据。清理后下次重新构建时需要再次下载依赖，请按需执行。
To support the development of my project, I plan to apply for a free VPS offered by ZMTO. My project currently involves Docker image support for multiple architectures (arm64 and amd64), as well as automation for building and pushing. Therefore, I am requesting a 4-core, 8GB RAM Ubuntu VPS with root access.

Thank you to the ZMTO team for your support. I look forward to leveraging this VPS to optimize my project's performance and development efficiency. If you have any questions or suggestions regarding my project, feel free to open an issue, and I will do my best to improve and optimize it.

Thank you for your attention and support!

Feel free to adjust any details as needed!

## Stargazers over time
[![Stargazers over time](https://starchart.cc/gooaclok819/sublinkX.svg?variant=adaptive)](https://starchart.cc/gooaclok819/sublinkX)

