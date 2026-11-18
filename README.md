# ima-mcp-server

用 Go 实现的腾讯 IMA MCP Server。基于 IMA Skill `1.1.10` 中公开描述的 OpenAPI、笔记图片流程和知识库文件上传流程实现，目标是以单二进制方式提供稳定、可复用的 MCP 工具。

> 本项目不是腾讯官方项目。使用前需要自行在 IMA 获取 OpenAPI Client ID 与 API Key。

## 功能

- 笔记：搜索、列表、读取、新建、追加、笔记本列表。
- 笔记图片：支持 Markdown 中的 PNG/JPEG/WebP；本地绝对路径、`file://`、data URL、公开 HTTP(S) URL；每张原始图片必须 **小于 3 MiB**。
- 知识库：详情、浏览、搜索、知识库搜索、可写知识库列表、重名检查、URL 导入、媒体信息。
- 文件上传：服务端强制执行 **预检 → 重名检查 → create_media → COS 上传 → add_knowledge**，不会让模型跳过安全门。
- URL 高级导入：HTML/微信文章走 `import_urls`；PDF/Word/PPT/Excel/Markdown/图片/音频等文件型 URL 自动下载、预检并走完整文件上传流程。
- MCP 传输：`stdio` 与 Streamable HTTP。
- 跨平台单二进制：Linux/macOS/Windows，amd64/arm64。

## 快速开始

### 1. 配置 IMA 凭证

从 IMA Agent Interface 获取 Client ID 和 API Key，然后任选一种方式配置。

环境变量：

```bash
export IMA_OPENAPI_CLIENTID="your_client_id"
export IMA_OPENAPI_APIKEY="your_api_key"
```

也兼容：

```bash
export IMA_CLIENT_ID="your_client_id"
export IMA_API_KEY="your_api_key"
```

或使用与原 Skill 兼容的配置文件：

```bash
mkdir -p ~/.config/ima
printf '%s' 'your_client_id' > ~/.config/ima/client_id
printf '%s' 'your_api_key' > ~/.config/ima/api_key
```

凭证默认只发送到 `https://ima.qq.com`。自定义 `IMA_BASE_URL` 会被拒绝；仅测试场景显式设置 `IMA_ALLOW_CUSTOM_BASE_URL=1` 才会放行。

### 2. 编译

本项目使用官方 MCP Go SDK `v1.8.0`，需要 Go 1.25+：

```bash
go build -o ima-mcp-server ./cmd/ima-mcp-server
```

或：

```bash
make build
```

### 3. 以 stdio 方式使用

```bash
./ima-mcp-server
```

典型 MCP 客户端配置：

```json
{
  "mcpServers": {
    "ima": {
      "command": "/absolute/path/to/ima-mcp-server",
      "args": ["-transport", "stdio"],
      "env": {
        "IMA_OPENAPI_CLIENTID": "your_client_id",
        "IMA_OPENAPI_APIKEY": "your_api_key"
      }
    }
  }
}
```

### 4. Streamable HTTP

默认只监听本机：

```bash
./ima-mcp-server -transport http -listen 127.0.0.1:8080
```

MCP endpoint：`http://127.0.0.1:8080/mcp`  
健康检查：`http://127.0.0.1:8080/healthz`

HTTP 模式本身没有额外身份认证，并使用服务端配置的一组 IMA 凭证。**不要直接暴露到公网或不受信任网络。** 如需远程使用，应在反向代理层增加 TLS、认证与访问控制。

## MCP Tools

| Tool | 作用 |
| --- | --- |
| `ima_search_notes` | 按标题/正文搜索笔记 |
| `ima_list_notes` | 列出全部或指定笔记本中的笔记 |
| `ima_get_note` | 获取笔记正文 |
| `ima_create_note` | 新建 Markdown 笔记 |
| `ima_append_note` | 向明确指定的已有笔记追加 Markdown |
| `ima_append_note_markdown` | 追加包含本地/远程图片的混合 Markdown；先完整预检图片再分批写入 |
| `ima_list_notebooks` | 列出笔记本 |
| `ima_get_knowledge_bases` | 获取 1-20 个知识库详情 |
| `ima_list_knowledge` | 浏览知识库/文件夹 |
| `ima_search_knowledge` | 搜索单个知识库内的文件和文件夹 |
| `ima_search_knowledge_bases` | 按名称搜索知识库；空 query 可用于列表 |
| `ima_list_addable_knowledge_bases` | 列出当前用户可添加内容的知识库 |
| `ima_check_duplicate_files` | 批量检查文件重名 |
| `ima_import_urls` | 导入网页/微信公众号文章 |
| `ima_get_media_info` | 获取媒体访问元数据；笔记类型会返回关联 note ID |
| `ima_add_note_to_knowledge_base` | 将已有笔记关联到知识库 |
| `ima_preflight_file` | 只做本地文件类型/大小/图片签名预检 |
| `ima_upload_file` | 完整、安全的本地文件上传流程 |
| `ima_add_url` | 自动判断网页 URL 与文件 URL，并执行正确流程 |

`create_media` 和文件型 `add_knowledge` 没有单独暴露成 MCP Tool，而是封装在 `ima_upload_file` / `ima_add_url` 内部，以保证安全门不可被跳过。

## 重名策略

文件上传默认 `duplicate_policy="error"`：检测到同名文件后返回：

```json
{
  "status": "duplicate",
  "duplicate": true,
  "file_name": "report.pdf"
}
```

此时应让用户选择：取消，或保留两份。用户确认保留两份后，重试并设置：

```json
{
  "duplicate_policy": "keep_both"
}
```

服务端会追加 `_YYYYMMDDHHmmss`，再次做重名检查后才继续上传。**不支持覆盖已有文件。**

## 文件支持与大小限制

与 Skill `1.1.10` 保持一致：

- Excel / CSV / Markdown / TXT / XMind / HTML：最大 10 MiB
- PNG / JPEG / WebP：最大 30 MiB
- EPUB：最大 50 MiB
- PDF / Word / PPT / 音频等：最大 200 MiB
- 视频文件不支持此上传流程
- 笔记内图片：每张原始文件必须严格小于 3 MiB

图片会检查真实 PNG/JPEG/WebP 文件签名，声明 MIME、扩展名和实际签名不一致时直接拒绝。

## 安全设计

1. **IMA 凭证边界**：Client ID/API Key 只用于 IMA OpenAPI 请求，不发送给 COS 或第三方 URL。
2. **COS 临时凭证**：只使用 `create_media` 返回的临时凭证完成当次 PUT；不写日志、不持久化。
3. **上传失败即停**：COS PUT 失败后不会调用 `add_knowledge`。
4. **SSRF 防护**：`ima_add_url` 与远程笔记图片拒绝 loopback、私网、link-local、multicast、unspecified、CGNAT 等目标；重定向也重新校验。
5. **远程图片不带 IMA 凭证**。
6. **本地文件权限**：stdio 模式下，MCP Server 与客户端运行在同一用户权限边界；模型传入的 `path` 能读取什么取决于该进程本身的操作系统权限。建议以最小权限用户运行。
7. **HTTP 模式**：默认绑定 `127.0.0.1`，没有内建多用户认证，不应裸露公网。

## 配置

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `IMA_OPENAPI_CLIENTID` | - | Client ID，优先级最高 |
| `IMA_OPENAPI_APIKEY` | - | API Key，优先级最高 |
| `IMA_CLIENT_ID` | - | 兼容变量 |
| `IMA_API_KEY` | - | 兼容变量 |
| `IMA_HTTP_TIMEOUT` | `60s` | IMA OpenAPI 请求超时，例如 `30s`、`2m` |
| `IMA_OPENAPI_CTX` | `skill_version=1.1.10` | 发给 IMA 的 `ima-openapi-ctx` |
| `IMA_BASE_URL` | `https://ima.qq.com` | IMA API Base URL；自定义默认被拒绝 |
| `IMA_ALLOW_CUSTOM_BASE_URL` | `0` | 测试环境允许自定义 Base URL |
| `IMA_ALLOW_INSECURE_BASE_URL` | `0` | 测试环境允许自定义 HTTP Base URL |

凭证查找优先级：环境变量 → `~/.config/ima/client_id` / `api_key`。

## 开发

```bash
make fmt
make test
make vet
make build
```

测试重点覆盖 IMA 请求协议、API 错误、文件预检、图片签名、重名门、上传事务顺序与 COS 签名逻辑。

## Release

创建版本 tag 后，GitHub Actions 使用 GoReleaser 构建：

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

构建时设置 `CGO_ENABLED=0`，方便分发单文件二进制。

## License

Apache-2.0
