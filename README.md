# PSL Test Logs MCP

部署在 PSL 测试机上的只读日志 MCP。它只读取配置文件中明确列出的日志 glob，
不接受客户端文件路径，不调用 Shell、Docker、`journalctl` 或远程接口。

## MCP 工具

### `list_test_log_sources`

列出允许查询的日志源名称，不暴露服务器绝对路径。

### `search_test_logs`

输入示例：

```json
{
  "source": "gk-user",
  "query": "816220425",
  "limit": 200,
  "case_sensitive": false
}
```

- `source` 必须来自服务端配置白名单。
- `query` 是字面量，不作为正则或 Shell 表达式；留空时返回最近日志行。
- 每个文件从尾部读取，结果按新到旧返回。
- 服务端统一限制文件数、总扫描字节、结果条数、单行长度和总输出大小。
- 自动脱敏 Authorization、token、password、secret、cookie、session、手机号等常见敏感值。
- 审计日志只记录 source、查询哈希、扫描量、结果数和耗时，不记录查询原文。

## 配置

示例配置已根据工作区各 `psl-be-gk-*` 仓库的 `deploy/supervisor_dev`
文件整理，覆盖 `activity`、`app`、`broker`、`commodity`、`external`、
`game`、`gift`、`im`、`room`、`user`、`wallet` 和 `worker`，日志根目录为
`/home/ecs-user/log`。`all-gk` 聚合源只组合上述明确前缀，不会读取该目录下
其他项目的日志。

配置示例见 `config/sources.example.json`。Supervisor 配置存在点号和下划线两种
历史文件名前缀，示例已分别覆盖；worker 还包含 `profile_cache` 和 `room_cache`
日志。所有 pattern 必须是绝对路径，不能包含 `..`。启动时会解析每个 pattern
的固定根目录；匹配到的符号链接若逃出根目录会被忽略。

建议以独立低权限用户运行，并只给该用户目标日志目录的读取权限。不要以 root 运行。

## 构建

SDK v1.7.0 要求 Go 1.25：

```bash
make test
make build-linux
```

Linux amd64 产物：

```text
bin/psl-test-logs-mcp-linux-amd64
```

## 部署到 005 测试机

目标机器：

```text
005: ps-sg-dev-002
IP: 192.168.35.220
用户: ecs-user
```

下面步骤沿用 GK 服务现有的 `gitroot`、`webroot` 和 `/home/ecs-user/log`
目录，不需要 root 权限，也不需要部署 Supervisor 常驻进程。MCP 会在 Codex
建立连接时启动，连接结束后退出。

### 1. 登录 005

在本机终端执行：

```bash
jump
```

进入 GateShell 后输入 `:5` 并回车，确认终端显示的机器 IP：

```bash
hostname -I
```

输出必须包含 `192.168.35.220`。如果不是该 IP，停止后续操作。

### 2. 确认日志和 Go 环境

```bash
go version
find /home/ecs-user/log -maxdepth 1 -type f -name '*.log' | head -n 20
```

MCP SDK 要求 Go 1.25。若 `go version` 低于 1.25，构建时使用
`GOTOOLCHAIN=auto`；测试机需要能够下载 Go 1.25 工具链。日志列表为空时先检查
当前机器是否部署了相应 GK 服务，不要放宽配置为 `/home/ecs-user/log/*.log`。

### 3. 拉取代码

首次部署：

```bash
mkdir -p /home/ecs-user/gitroot
cd /home/ecs-user/gitroot
git clone git@github.com:zhengjinwei-ola/psl-test-logs-mcp.git
cd psl-test-logs-mcp
```

若仓库已存在：

```bash
cd /home/ecs-user/gitroot/psl-test-logs-mcp
git pull --ff-only origin main
```

### 4. 测试并构建

```bash
cd /home/ecs-user/gitroot/psl-test-logs-mcp
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto make build
```

构建产物应为 Linux x86-64 可执行文件：

```bash
file bin/psl-test-logs-mcp
```

### 5. 安装二进制和配置

```bash
mkdir -p /home/ecs-user/webroot/psl-test-logs-mcp/bin
mkdir -p /home/ecs-user/webroot/psl-test-logs-mcp/configs

install -m 0750 \
  bin/psl-test-logs-mcp \
  /home/ecs-user/webroot/psl-test-logs-mcp/bin/psl-test-logs-mcp

install -m 0640 \
  config/sources.example.json \
  /home/ecs-user/webroot/psl-test-logs-mcp/configs/sources.json
```

确认安装结果和日志读取权限：

```bash
test -x /home/ecs-user/webroot/psl-test-logs-mcp/bin/psl-test-logs-mcp
test -r /home/ecs-user/webroot/psl-test-logs-mcp/configs/sources.json
test -r /home/ecs-user/log
```

### 6. 远端启动冒烟

stdio MCP 从标准输入读取协议；使用空输入启动时应正常退出且不输出 fatal 错误：

```bash
/home/ecs-user/webroot/psl-test-logs-mcp/bin/psl-test-logs-mcp \
  --config=/home/ecs-user/webroot/psl-test-logs-mcp/configs/sources.json \
  </dev/null

echo $?
```

退出码应为 `0`。如果提示 pattern root 不存在，说明配置中的
`/home/ecs-user/log` 与机器不一致，应先核对部署和 Supervisor 配置，不要创建
虚假目录掩盖问题。

### 7. 在本地 Codex 注册 MCP

本地启动命令复用 `psl-devtools-mcp` 中的 005 GateShell 启动器：

```json
{
  "mcpServers": {
    "psl-test-logs": {
      "command": "/Users/oswin/ola/psl-devtools-mcp/scripts/run-ps-sg-dev-002-mcp.exp",
      "args": [
        "/home/ecs-user/webroot/psl-test-logs-mcp/bin/psl-test-logs-mcp",
        "--config=/home/ecs-user/webroot/psl-test-logs-mcp/configs/sources.json"
      ]
    }
  }
}
```

修改 MCP 配置后需要重启或重载 Codex 会话。

重新加载后，应能发现：

```text
mcp__psl_test_logs__list_test_log_sources
mcp__psl_test_logs__search_test_logs
```

先调用 `list_test_log_sources`，确认返回 `gk-activity`、`gk-user`、`gk-room`
等 source，再执行一条小范围查询，例如 source 为 `gk-user`、query 为某个测试
UID、limit 为 `20`。

### 8. 后续升级

```bash
cd /home/ecs-user/gitroot/psl-test-logs-mcp
git pull --ff-only origin main
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto make build
install -m 0750 bin/psl-test-logs-mcp \
  /home/ecs-user/webroot/psl-test-logs-mcp/bin/psl-test-logs-mcp
install -m 0640 config/sources.example.json \
  /home/ecs-user/webroot/psl-test-logs-mcp/configs/sources.json
```

MCP 不是常驻进程，不需要重启 Supervisor；关闭现有 Codex 会话并重新加载即可
使用新二进制。若需要回滚，检出此前确认可用的 Git commit，重新执行测试、构建
和 `install`，不要删除业务日志。

## 安全边界

- 只支持 stdio MCP，不开放网络端口。
- 不提供日志删除、文件写入、Shell 执行、服务重启或配置修改。
- 客户端不能指定路径，只能选择预配置 source。
- 忽略目录、设备、Socket 和逃出白名单根目录的符号链接。
- 远端 stderr 仅用于本机审计；通过 GateShell 启动时会被启动器丢弃，避免污染协议流。
