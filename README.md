# OpenAI Proxy

基于 Go 实现的 OpenAI API HTTP 代理，支持多目标路由和流式响应。

## 功能特性

- **多目标代理**: 支持通过配置文件或命令行同时代理多个目标服务
- **Host 路由**: 根据请求 Host 自动路由到对应的目标服务
- **流式响应**: 完美支持 ChatGPT 流式响应
- **跨平台**: 支持 Linux/macOS/Windows，适配云函数部署
- **零依赖**: 仅使用 Go 标准库

## 快速开始

### 安装运行

```bash
# 编译
make

# 运行（默认监听 9001 端口）
./openai-proxy

# 指定监听端口
./openai-proxy -listen=:9000

# 指定代理目标
./openai-proxy -targets=openai:api.openai.com
```

### 使用环境变量

```bash
# 设置监听地址
export OPENAI_PROXY_LISTEN=:9000

# 设置代理目标（支持多个，逗号分隔，key 是子域名，value 是目标域名）
export OPENAI_PROXY_TARGETS=oa:api.openai.com,azure:your-azure.openai.azure.com

# 运行
./openai-proxy
```


## 多目标路由配置

### 配置文件方式

```bash
# 格式: key:domain
# - key: 自定义域名的子域名（如 oa、azure 等）
# - domain: 代理目标域名（代码会自动添加 https:// 前缀）
./openai-proxy -targets=oa:api.openai.com,azure:your-azure.openai.azure.com
```

例如：
- 配置 `oa:api.openai.com`
- 访问 `oa.mydomain.com` → 转发到 `https://api.openai.com`

### Host 路由规则

请求会根据 Host 前缀自动路由到对应目标：

```
# 配置: -targets=oa:api.openai.com,azure:your-azure.openai.azure.com

# 请求 oa.mydomain.com → 路由到 https://api.openai.com
# 请求 azure.mydomain.com → 路由到 https://your-azure.openai.azure.com

curl -X POST https://oa.mydomain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

### 请求头方式覆盖

可以通过 `X-Target-Host` 请求头临时指定目标（优先级最高）：

```bash
curl -X POST https://proxy.example.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Target-Host: api.openai.com" \  # 不需要写 https://，会自动添加
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

## 配置选项

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-listen` | `:9001` | 监听地址和端口 |
| `-targets` | - | 目标映射表，格式: `key:domain.com` |

### 环境变量

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `OPENAI_PROXY_LISTEN` | `:9001` | 监听地址 |
| `OPENAI_PROXY_TARGETS` | - | 目标映射表 |
| `ENV` | - | 设置为 `local` 启用本地测试模式 |
| `HTTP_PROXY` / `http_proxy` | - | HTTP 代理地址（本地模式） |

### 本地测试模式

设置 `ENV=local` 可启用本地代理测试：

```bash
ENV=local ./openai-proxy
```

此时会使用默认代理 `http://127.0.0.1:10809` 进行测试。

## 编译部署

### 手动编译

```bash
# 编译本地版本
make

# 编译指定平台版本
make dist/linux_amd64/openai-proxy
make dist/darwin_arm64/openai-proxy

# 编译所有平台版本
make dist
```

### 云函数部署

```bash
# 编译 Linux AMD64 版本
GOOS=linux GOARCH=amd64 go build -o main main.go

# 打包
zip openai-proxy.zip main scf_bootstrap
```

部署到腾讯云函数、阿里云函数或 AWS Lambda 时，上传打包好的 zip 文件即可。

### Nginx 配置（流式响应支持）

```nginx
location / {
    proxy_pass http://127.0.0.1:9000;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## 请求示例

### ChatGPT 流式调用

```bash
curl -X POST https://your-proxy.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "讲个笑话"}],
    "stream": true
  }'
```

### Azure OpenAI

```bash
curl -X POST https://azure.your-proxy.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "api-key: $AZURE_API_KEY" \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 技术规格

| 项目 | 规格 |
|------|------|
| Go 版本 | 1.17+ |
| 并发模型 | Go goroutine |
| 连接池 | MaxIdleConns: 100, MaxIdleConnsPerHost: 20 |
| 空闲超时 | 90秒 |
| 流式缓冲区 | 1024字节 |
| 日志格式 | `[PROXY] YYYY/MM/DD HH:MM:SS message` |

## License

MIT
