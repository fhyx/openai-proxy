GOOS=linux GOARCH=amd64 go build -o openai-proxy main.go
zip openai-proxy.zip openai-proxy scf_bootstrap

