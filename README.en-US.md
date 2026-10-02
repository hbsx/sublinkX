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
    <div align="center"> <a href="README.md">中文<a> | English</div>
</div>
## [Project Information]

Project based on sublink project secondary development: https://github.com/jaaksii/sublink

Front-end based on: https://github.com/youlaitech/vue3-element-admin

Backend using go+gin+gorm

Default account admin password 123456 self-modification

Because of the rewrite there are still a lot of layout structure and a little less functionality

## [Project Features]

* Manage subscriptions, node groups, access records, and templates from one web console.
* Install with a Linux binary and upgrade menu, or build with Docker. Data directories can be mounted separately for backup and migration.
* Generate v2ray Base64 subscriptions, Clash YAML, and Surge configurations. Incompatible nodes are filtered for the selected client.
* The latest branch removes the login captcha. Change the default administrator password immediately after first use.

### Supported node protocols

| Protocol | Parse subscription | Clash output | Surge output |
| --- | --- | --- | --- |
| Shadowsocks (SS) / ShadowsocksR (SSR) | Yes | Yes | SS only |
| VMess | Yes | Yes | Yes |
| VLESS (including XHTTP) | Yes | Yes | Filtered when unsupported |
| Trojan | Yes | Yes | Yes |
| Hysteria / Hysteria2 | Yes | Yes | Hysteria2 only |
| TUIC | Yes | Yes | Yes |
| AnyTLS | Yes | Yes | Filtered when unsupported |
| SOCKS5, HTTP, and HTTPS proxy URLs | Yes | Yes | Filtered when unsupported |

Actual availability also depends on the receiving client's version. Unsupported nodes are filtered rather than emitted as unusable configuration.

## [Project Preview]

![1712594176714](webs/src/assets/1.png)
![1712594176714](webs/src/assets/2.png)

## Release notes

### [2.1.2](https://github.com/hbsx/sublinkX/releases/tag/2.1.2) · 2026-10-02

This release fixes subscription isolation, account security, templates, and installer updates, and includes the current branch's fixes in Docker images and release binaries. See [RELEASE_NOTES.md](RELEASE_NOTES.md) for the full changelog in Chinese.

#### Subscriptions and account security

- Isolate concurrent subscription requests so one request cannot read another subscription's nodes.
- Replace predictable name-based MD5 subscription links with persisted random credentials. Renaming a subscription or restarting normally does not rotate these credentials.
- Revoke previous login tokens after password changes, account resets, or logout. A login verified against the old password cannot acquire a valid new session during a concurrent password change.
- Store passwords as bcrypt hashes and migrate legacy plaintext passwords automatically. Existing credentials still work when logging in again.
- Remove sensitive password, node URL, and subscription credential logging. Existing historical logs are not deleted automatically.

#### Proxy detection and network requests

- Stop treating HTTP/HTTPS proxies as remote subscriptions. The node editor offers automatic detection, proxy node, and remote subscription choices.
- Automatic detection treats HTTP/HTTPS URLs with user credentials, a name fragment, or only an explicit port as proxies. Select proxy node for an ambiguous proxy using the default port, and remote subscription for ambiguous subscription URLs.
- Bound subscription, template, and IP lookup requests with timeouts, response status checks, and content size limits.

#### Templates and command-line behavior

- Preserve Surge sections and rules without duplicating them.
- Return errors for invalid Clash proxy group structures, and continue processing later groups after a relay group.
- Parse ports as decimal, including leading zeros: `08000` means `8000`. Reject invalid ports.
- Keep `--version` and `healthcheck` free of directory and application data creation. Health checks also verify the running service version.

#### Docker, installer, and release assets

- Docker images `ghcr.io/hbsx/sublinkx:2.1.2` and `latest` support Linux AMD64, ARMv7, and ARM64.
- Release assets include Linux AMD64/ARM64 and Windows AMD64 binaries plus `SHA256SUMS`, downloaded from this fork.
- Linux installation and menu updates verify downloads, stop the service, back up the program and data, and restore them if startup verification fails. Failed port changes also restore the previous state.
- Keep this branch's AnyTLS, SOCKS5, HTTP/HTTPS, VLESS XHTTP support, and login without a captcha.

#### Upgrade notes

1. Back up existing data first. The database migrates automatically and preserves nodes, subscription names, configuration, and templates; do not delete it.
2. **Log in again and copy new subscription URLs from the dashboard. Old MD5 links no longer work; replace saved links in your clients.**
3. Pull the new Docker image and recreate the container with the original `db`, `template`, and `logs` mounts. Restarting an old container does not upgrade it. Docker upgrades do not include the Linux installer's automatic rollback.
4. For Linux binary deployments, rerun the one-command installer below to update both the program and menu while preserving existing credentials, data, and ports.

Go tests, race detection, static checks, frontend build, ARM64 startup/restart and legacy migration, and installer rollback/checksum checks passed.

### 2.1 (historical release)

- Added node grouping, updated the node management page, and fixed subscription parsing.

## [Installation instructions]

The latest release in this fork includes AnyTLS, SOCKS5, HTTP/HTTPS, and VLESS XHTTP support, and removes the login captcha.

### linux method:
```
curl -fsSL -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/hbsx/sublinkX/main/install.sh | sudo bash
```

```sublink``` Calls out the menu.

Then just type in the install script

### Docker

Docker Compose is recommended:

```bash
git clone https://github.com/hbsx/sublinkX.git
cd sublinkX
docker compose up -d
```

Compose pulls the matching AMD64, ARMv7, or ARM64 image built by this repository. Persistent data is stored in `data/db`, `data/template`, and `data/logs`. The configuration uses `restart: unless-stopped`, so the container starts again after Docker or the server restarts.

Copy `.env.example` to `.env` to change the host port, pin an image version, or reuse existing data. `SUBLINK_PORT` changes only the host port; keep `port: 8000` in the container's `db/config.yaml`.

Before migrating an existing container, inspect its mounts with `docker inspect sublinkx --format '{{json .Mounts}}'` and back up the data. Set `SUBLINK_DATA_DIR` in `.env` to the existing parent directory containing `db`, `template`, and `logs`. Stop and remove the old container before starting Compose. Using a different data directory starts a new, empty database.

Containers stopped manually remain stopped under `unless-stopped`; start them explicitly to resume. An `unhealthy` status alone does not make Docker restart the container.

To update a Compose deployment:

```bash
cd sublinkX
git pull --ff-only
docker compose pull
docker compose up -d
```

To use `docker run` instead:

```bash
mkdir -p ~/sublinkx/db ~/sublinkx/template ~/sublinkx/logs
cd ~/sublinkx
docker pull ghcr.io/hbsx/sublinkx:latest
docker run -d \
  --name sublinkx \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/db:/app/db" \
  -v "$PWD/template:/app/template" \
  -v "$PWD/logs:/app/logs" \
  ghcr.io/hbsx/sublinkx:latest
```

For a container created with the older instructions, add the restart policy without recreating it:

```bash
docker update --restart unless-stopped sublinkx
```

To support the development of my project, I plan to apply for a free VPS offered by ZMTO. My project currently involves Docker image support for multiple My project currently involves Docker image support for multiple architectures (arm64 and amd64), as well as automation for building and pushing. Therefore, I am requesting a 4-core, 8GB RAM Ubuntu VPS with root access.

Thank you to the ZTMO team for your support. I look forward to leveraging this VPS to optimize my project's performance and development efficiency. have any questions or suggestions regarding my project, feel free to open an issue, and I will do my best to improve and optimize it.

Thank you for your attention and support!

Feel free to adjust any details as needed!

## Stargazers over time
[![Stargazers over time](https://starchart.cc/gooaclok819/sublinkX.svg?variant=adaptive)](https://starchart.cc/gooaclok819/sublinkX)

