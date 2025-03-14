# GO-OPENAI-PROXY

基于 Go 实现的 OpenAI API HTTP 代理

> 想要快速体验，将 OpenAI API 调用域名从默认的 `api.openai.com` 调整为 `proxy.geekai.co` 即可。你可以在这里预览演示效果：[演示应用](https://geekai.co/dati?invite_code=S564yq)。

### 切换到 Azure OpenAI

默认在 9000 端口代理 OpenAI API，要想切换到 Azure OpenAI API，可以在 `scf_bootstrap` 的启动命令中添加域名参数来指定你的 Azure OpenAI API Endpoint:

```bash
./dist/linux_amd64/openai-proxy -target=your-azure-openai-endpoint
```

如果 9000 端口被占用，可以通过 `-listen=:9001` 指定其他端口。

### 也可使用环境变量来设定参数

```plain
OPENAI_PROXY_LISTEN=:9000
OPENAI_PROXY_TARGET=https://api.openai.com
```

```bash
OPENAI_PROXY_LISTEN=:1234 ./openai-proxy
```

### 代理任意全球域名

这个工具最早是为 OpenAI 代理而生，但实际上现在已经可以支持通过一个入口代理任意域名，只需要在发起发起代理请求的时候通过 `X-Target-Host` 设置你想要代理的域名（不带 `http(s)://` 前缀）即可，优先级是`请求头>命令行参数>默认值`：

```go
req.Header.Set("x-target-host", "api.open.ai")
```

### 手动编译

编译成本地版
```bash
make
```
或直接编译成Linux版
```bash
make dist/linux_amd64/openai-proxy
```

### 编译打包（限Linux）

你可以修改源代码调整代理逻辑，然后编译打包进行部署：

```bash
./build.sh
```

此命令需要本地安装[go开发环境](https://go.dev/)，如果不想本地安装 go 环境进行编译打包，可以直接下载根据最新源代码编译打包好的 `openai-proxy.zip`：[Releases](https://github.com/geekr-dev/openai-proxy/releases)

### 部署测试

> 支持部署到腾讯/阿里云函数、AWS lambda 函数以及任意云服务器，以下以腾讯云函数为例进行演示。

然后在腾讯云云函数代码管理界面上传打包好 zip 包即可完成部署：

![](https://image.gstatics.cn/2023/03/06/image-20230306171340547.png)

你可以通过腾讯云云函数提供的测试工具进行测试，也可以本地通过 curl/postman 进行测试，使用的时候只需要将 `api.openai.com` 替换成代理域名 `proxy.geekai.co` 即可：
 
![](https://geekr.gstatics.cn/wp-content/uploads/2023/03/image-38.png)

你可以选择自己搭建，也可以直接使用我提供的代理域名 `proxy.geekai.co`，反正是免费的。关于代理背后的原理，可以看我在极客书房发布的这篇教程：[国内无法调用 OpenAI 接口的解决办法](https://geekr.dev/posts/chatgpt-website-by-laravel-10#toc-5)。

本地调试走VPN的话可以设置环境变量 `ENV=local`，然后直连 `api.openai.com`：

```go
// 本地测试通过代理请求 OpenAI 接口
if os.Getenv("ENV") == "local" {
    proxyURL, _ := url.Parse("http://127.0.0.1:10809")
    client.Transport = &http.Transport{
        Proxy:           http.ProxyURL(proxyURL),
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
}
```
### 流式响应支持

这个源代码本身是支持 stream 流式响应代理的，但目前很多云函数并不支持分块流式传输。所以，如果你需要实现流式响应，可以把编译后的二进制文件 `main` 丢到任意海外云服务器运行，这样就变成支持流式响应的 OpenAI HTTP 代理了，如果你不想折腾，可以使用我这边提供的 `proxy.geekai.co` 作为代理进行测试：

<img width="965" alt="image" src="https://user-images.githubusercontent.com/114386672/225609817-ca5c106b-22d4-4ae9-b3df-ca2c46d56843.png">

如果你是通过 Nginx 这种反向代理对外提供服务，记得通过如下配置项将默认缓冲和缓存关掉才会生效：

```
proxy_buffering off;
proxy_cache off;
```
