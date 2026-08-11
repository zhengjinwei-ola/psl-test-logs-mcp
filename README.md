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

部署配置：

```bash
sudo install -d -m 0750 /etc/psl-test-logs-mcp
sudo install -m 0640 config/sources.example.json /etc/psl-test-logs-mcp/sources.json
```

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

本仓库只准备部署产物，不自动写入测试机。上传二进制和配置后，可放置为：

```text
/opt/psl-test-logs-mcp/bin/psl-test-logs-mcp
/etc/psl-test-logs-mcp/sources.json
```

远端手工验证：

```bash
/opt/psl-test-logs-mcp/bin/psl-test-logs-mcp \
  --config=/etc/psl-test-logs-mcp/sources.json
```

本地 Codex MCP 启动命令可复用 `psl-devtools-mcp` 中的 005 启动器：

```json
{
  "mcpServers": {
    "psl-test-logs": {
      "command": "/Users/oswin/ola/psl-devtools-mcp/scripts/run-ps-sg-dev-002-mcp.exp",
      "args": [
        "/opt/psl-test-logs-mcp/bin/psl-test-logs-mcp",
        "--config=/etc/psl-test-logs-mcp/sources.json"
      ]
    }
  }
}
```

修改 MCP 配置后需要重启或重载 Codex 会话。

## 安全边界

- 只支持 stdio MCP，不开放网络端口。
- 不提供日志删除、文件写入、Shell 执行、服务重启或配置修改。
- 客户端不能指定路径，只能选择预配置 source。
- 忽略目录、设备、Socket 和逃出白名单根目录的符号链接。
- 远端 stderr 仅用于本机审计；通过 GateShell 启动时会被启动器丢弃，避免污染协议流。
