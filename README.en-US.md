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

## [Updated Description]

####Backend Update

1. Fix and refactor a large number of Node templates and the underlying code for new groupings

2. Add grouping functionality to nodes

3. Fix bug that subscription resolution is empty or etc

####Front-end update

1. Refactor front-end node page to add grouping function (temporarily only some simple functions)

## [Installation instructions]

The latest release in this fork includes AnyTLS, SOCKS5, HTTP/HTTPS, and VLESS XHTTP support, and removes the login captcha.

### linux method:
```
curl -s -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/hbsx/sublinkX/main/install.sh | sudo bash
```

```sublink``` Calls out the menu.

Then just type in the install script

### docker method:

Create a directory where you want it to be located, such as mkdir sublinkx.

Then cd into the directory and enter the following command to mount the data.

All you need to back up is the db and templates.
```
docker build -t hbsx/sublinkx:latest https://github.com/hbsx/sublinkX.git

docker run --name sublinkx -p 8000:8000 \
-v $PWD/db:/app/db \
-v $PWD/template:/app/template \
-v $PWD/logs:/app/logs \
-d hbsx/sublinkx:latest
```

To support the development of my project, I plan to apply for a free VPS offered by ZMTO. My project currently involves Docker image support for multiple My project currently involves Docker image support for multiple architectures (arm64 and amd64), as well as automation for building and pushing. Therefore, I am requesting a 4-core, 8GB RAM Ubuntu VPS with root access.

Thank you to the ZTMO team for your support. I look forward to leveraging this VPS to optimize my project's performance and development efficiency. have any questions or suggestions regarding my project, feel free to open an issue, and I will do my best to improve and optimize it.

Thank you for your attention and support!

Feel free to adjust any details as needed!

## Stargazers over time
[![Stargazers over time](https://starchart.cc/gooaclok819/sublinkX.svg?variant=adaptive)](https://starchart.cc/gooaclok819/sublinkX)

