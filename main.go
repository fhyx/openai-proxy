// Package main provides a simple HTTP proxy for OpenAI API requests.
// It can also be used to proxy requests to any domain by setting the X-Target-Host header.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Target    string // 目标域名
	Listen    string // 监听端口
	LocalEnv  bool   // 是否本地环境
	HttpProxy string // 本地代理地址和端口
}

// Proxy represents the HTTP proxy server
type Proxy struct {
	config     Config
	httpClient *http.Client
	logger     *log.Logger
}

// NewProxy creates a new proxy instance with the given configuration
func NewProxy(config Config) *Proxy {
	// 创建HTTP客户端
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	// 如果是本地环境且配置了HTTP代理，则使用代理
	if config.LocalEnv && config.HttpProxy != "" {
		proxyURL, err := url.Parse(config.HttpProxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	client := &http.Client{
		Transport: transport,
		// 代理不干涉超时逻辑，由客户端自行设置
	}

	return &Proxy{
		config:     config,
		httpClient: client,
		logger:     log.New(os.Stderr, "[PROXY] ", log.LstdFlags),
	}
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	p.logger.Printf("Starting proxy server on %s, targeting %s", p.config.Listen, p.config.Target)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    p.config.Listen,
		Handler: http.HandlerFunc(p.handleRequest),
	}

	// 启动HTTP服务器
	return server.ListenAndServe()
}

// handleRequest handles incoming HTTP requests
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 创建请求上下文
	ctx := r.Context()

	// 处理请求
	if err := p.processRequest(ctx, w, r); err != nil {
		p.logger.Printf("Error processing request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// processRequest processes the incoming request and forwards it to the target
func (p *Proxy) processRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 验证URL
	if _, err := url.Parse(r.URL.String()); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 构建目标URL
	targetURL, err := p.buildTargetURL(r)
	if err != nil {
		return fmt.Errorf("failed to build target URL: %w", err)
	}

	// 本地环境打印代理请求URL
	if p.config.LocalEnv {
		p.logger.Printf("Proxying request to: %s", targetURL)
	}

	// 创建代理请求
	proxyReq, err := p.createProxyRequest(ctx, r, targetURL)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// 发送代理请求
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("failed to send proxy request: %w", err)
	}
	defer resp.Body.Close()

	// 处理响应
	return p.handleResponse(w, resp)
}

// buildTargetURL builds the target URL for the proxy request
func (p *Proxy) buildTargetURL(r *http.Request) (string, error) {
	// 去掉环境前缀（针对腾讯云，如果包含的话，目前只用到了test和release）
	path := strings.Replace(r.URL.Path, "/release", "", 1)
	path = strings.Replace(path, "/test", "", 1)

	// 构建目标URL
	// 优先级: X-Target-Host 头 > 配置的目标域名
	var targetURL string
	if targetHost := r.Header.Get("X-Target-Host"); targetHost != "" {
		targetURL = "https://" + targetHost + path
	} else {
		targetURL = p.config.Target + path
	}

	// 添加查询参数
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	return targetURL, nil
}

// createProxyRequest creates a new HTTP request to be sent to the target
func (p *Proxy) createProxyRequest(ctx context.Context, r *http.Request, targetURL string) (*http.Request, error) {
	// 创建新请求
	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		return nil, err
	}

	// 复制请求头
	p.copyHeaders(proxyReq.Header, r.Header)

	return proxyReq, nil
}

// copyHeaders copies HTTP headers from source to destination
func (p *Proxy) copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// handleResponse handles the response from the target server
func (p *Proxy) handleResponse(w http.ResponseWriter, resp *http.Response) error {
	// 复制响应头
	p.copyHeaders(w.Header(), resp.Header)

	// 设置响应状态码
	w.WriteHeader(resp.StatusCode)

	// 流式传输响应体
	return p.streamResponse(w, resp.Body)
}

// streamResponse streams the response body to the client
func (p *Proxy) streamResponse(w http.ResponseWriter, body io.ReadCloser) error {
	// 创建缓冲区
	buf := make([]byte, 1024)

	// 流式传输
	for {
		n, err := body.Read(buf)
		if err == io.EOF || n == 0 {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}

		if _, err = w.Write(buf[:n]); err != nil {
			return fmt.Errorf("error writing response: %w", err)
		}

		// 刷新响应
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// loadConfig loads the application configuration from command line flags and environment variables
func loadConfig() Config {
	var config Config

	// 从环境变量获取本地环境标志
	config.LocalEnv = os.Getenv("ENV") == "local"

	// 从环境变量获取HTTP代理
	config.HttpProxy = os.Getenv("HTTP_PROXY")
	if config.HttpProxy == "" {
		config.HttpProxy = os.Getenv("http_proxy")
	}

	// 如果没有设置HTTP代理且是本地环境，使用默认代理
	if config.HttpProxy == "" && config.LocalEnv {
		config.HttpProxy = "http://127.0.0.1:10809" // 默认本地代理
	}

	// 从命令行参数获取配置
	flag.StringVar(&config.Target, "target", envOr("OPENAI_PROXY_TARGET", "https://api.openai.com"),
		"The target domain to proxy.")
	flag.StringVar(&config.Listen, "listen", envOr("OPENAI_PROXY_LISTEN", ":9000"),
		"The proxy listen address.")
	flag.Parse()

	return config
}

// envOr returns the value of the environment variable or a fallback value
func envOr(key, fallback string) string {
	if s, ok := os.LookupEnv(key); ok && len(s) > 0 {
		return s
	}
	return fallback
}

func main() {
	// 加载配置
	config := loadConfig()

	// 创建代理服务器
	proxy := NewProxy(config)

	// 启动代理服务器
	if err := proxy.Start(); err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
}
