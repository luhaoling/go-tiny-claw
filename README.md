# go-tiny-claw

`go-tiny-claw` 是一个用 Go 编写的轻量级 AI Agent 示例项目。它通过飞书应用机器人接收单聊消息，将消息交给大模型处理，并可调用本地文件和命令行工具，最后把执行过程与回复发回飞书。

当前主程序使用智谱 GLM-4.5-Air，并通过 OpenAI 兼容接口调用。项目还包含一个 Anthropic SDK Provider 实现，供后续扩展；目前主程序没有使用它。

## 功能

- 通过飞书事件回调接收机器人单聊消息。
- 使用 ReAct 循环调用大模型和本地工具。
- 支持读取文件、写入文件、局部编辑文件和执行 Bash 命令。
- 只读工具调用可并发执行；如果一批调用中包含写操作或未知工具，则整批按顺序执行。
- 通过 Reporter 将思考状态、工具执行情况和模型回复发送回飞书。

## 项目结构

```text
cmd/claw/              服务入口：初始化模型、工具和飞书 HTTP 回调
internal/engine/       Agent 主循环、工具调用和 Reporter 接口
internal/feishu/       飞书事件分发、消息发送及 Agent 桥接
internal/provider/     大模型 Provider 接口及智谱兼容实现
internal/schema/       消息、工具调用和工具结果的数据结构
internal/tools/        工具注册表及文件、Bash 工具
```

## 环境要求

- Go `1.26.11`（见 `go.mod`）
- 可访问智谱 API 的网络环境
- 飞书开放平台中的应用机器人
- Linux/macOS 上使用 Bash 工具时需要安装 `bash`

## 配置

启动前设置以下环境变量：

| 变量 | 必需 | 用途 |
| --- | --- | --- |
| `ZHIPU_API_KEY` | 是 | 智谱 API 密钥 |
| `FEISHU_APP_ID` | 是 | 飞书应用的 App ID |
| `FEISHU_APP_SECRET` | 是 | 飞书应用的 App Secret |
| `FEISHU_VERIFY_TOKEN` | 否* | 飞书事件回调校验 Token |
| `FEISHU_ENCRYPT_KEY` | 否* | 飞书事件加密密钥；未配置加密时留空 |

`*` 如果飞书后台配置了校验 Token 或加密密钥，环境变量必须与后台一致。

Linux/macOS 示例：

```bash
export ZHIPU_API_KEY="替换为你的智谱 API Key"
export FEISHU_APP_ID="替换为你的飞书 App ID"
export FEISHU_APP_SECRET="替换为你的飞书 App Secret"
export FEISHU_VERIFY_TOKEN="替换为飞书后台的 Verification Token"
export FEISHU_ENCRYPT_KEY=""
```

不要把真实密钥提交到代码仓库。若写入 shell 配置文件，修改后可用 `source ~/.bashrc`（或对应 shell 配置文件）让当前终端加载。

## 飞书应用配置

在飞书开放平台配置应用机器人：

1. 添加事件 `im.message.receive_v1`（接收消息）。
2. 开通接收机器人单聊消息所需权限，例如 `im:message`；具体可用权限以应用后台显示为准。
3. 配置事件接收方式为**将事件发送至开发者服务器**。本项目使用 HTTP 回调，不包含飞书长连接接收客户端。
4. 设置回调 URL 为公网可访问的 HTTPS 地址，并使用路径 `/webhook/event`，例如 `https://bot.example.com/webhook/event`。
5. 完成回调地址校验，并按飞书后台要求发布应用或权限变更。
6. 将机器人添加到当前企业或可用范围，然后在飞书中向机器人发送单聊消息。

如果回调地址前面使用 Nginx、Caddy 等反向代理，请将 `/webhook/event` 转发到运行本项目的 `48080` 端口。

## 启动

从项目根目录运行：

```bash
go run ./cmd/claw
```

服务监听 `:48080`。启动成功时会在控制台打印监听日志。飞书事件到达时会打印 `[HTTP]` 日志；识别为消息事件后会打印 `[Feishu]收到会话...`。

如需编译二进制：

```bash
go build -o claw ./cmd/claw
./claw
```

## 使用 Docker 后台运行

项目提供 `Dockerfile` 和 `compose.yaml`。Compose 会构建镜像、在后台启动容器，并在 Docker 服务或主机重启后自动恢复运行。

在 Linux 项目目录执行：

```bash
cp .env.example .env
nano .env
mkdir -p workspace
CLAW_UID=$(id -u) CLAW_GID=$(id -g) docker compose up -d --build
```

将 `.env` 中的占位内容替换为真实凭证。`.env` 已加入 `.gitignore`，不要提交密钥。容器将 `workspace/` 挂载为 Agent 的工作目录，Agent 对该目录中文件的修改会保留在主机上。

常用管理命令：

```bash
docker compose logs -f claw   # 查看实时日志
docker compose restart claw   # 重启
docker compose stop claw      # 停止
docker compose up -d --build  # 重新构建并后台启动
```

飞书回调仍使用宿主机 `48080` 端口；请确保飞书配置的 HTTPS 回调 URL（路径 `/webhook/event`）经反向代理转发到该端口。不要把 `.env` 放入镜像或公开仓库。

## 工具行为

| 工具 | 作用 | 备注 |
| --- | --- | --- |
| `read_file` | 读取文件 | 返回内容最多 8000 字节 |
| `write_file` | 创建或覆盖文件 | 必须提供路径和完整内容 |
| `edit_file` | 对文件做局部文本替换 | 使用唯一文本片段匹配 |
| `bash` | 在工作目录运行 Bash 命令 | 单次最多运行 30 秒，输出最多 8000 字节 |

工作目录取程序启动时的当前目录。工具参数中的相对路径以该目录为基准。请只在可信环境中运行本项目：模型可请求写文件、编辑文件并执行 Bash 命令。

## 排查

- **没有启动日志**：确认命令在项目根目录执行、必需环境变量已设置，并查看启动该进程的终端或服务日志。
- **回调测试成功，单聊消息没有 `[HTTP]` 日志**：检查事件接收方式、`im.message.receive_v1` 订阅、机器人可用范围、消息权限、公网 URL 和反向代理/防火墙设置。回调测试只验证测试请求，不代表真实消息事件已投递。
- **有 `[HTTP]` 但没有 `[Feishu]收到会话...`**：检查飞书事件投递记录中的事件类型、投递状态，以及 Verify Token 和 Encrypt Key 是否与后台一致。
- **有消息日志但没有回复**：查看后续 `[Engine]` 日志和 `[Feishu]发送消息...` 日志；确认 `ZHIPU_API_KEY` 有效，并且应用拥有发送消息权限。
- **本机诊断 HTTP 服务**：在另一终端运行 `curl -i http://127.0.0.1:48080/webhook/event`。该 GET 请求没有飞书事件 JSON，可能返回解析错误；它只用于确认请求是否到达本地路由，不是消息事件模拟。

## 测试

运行项目测试：

```bash
go test ./...
```
