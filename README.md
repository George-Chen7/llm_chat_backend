## 项目简介
基于 Gin 的 LLM 对话后端骨架，包含配置加载、SSE 流式聊天接口与路由占位符。

## 目录结构
- `cmd/main.go`：入口，加载配置并启动服务
- `config.yaml`：示例配置（服务端口、调试开关、LLM 地址与密钥）
- `internal/config`：配置结构体与 YAML 加载
- `internal/router`：路由注册，包含登录、聊天、语音、管理模块及鉴权中间件占位
- `internal/handler`：聊天 SSE 处理器与各接口占位 Handler
- `internal/middlewares`：鉴权中间件占位

## 快速开始
1. 确保安装 Go 1.21+
2. 安装依赖（首次运行自动拉取）：`go run ./cmd/main.go`
3. 或先执行：`go mod tidy`
4. 运行服务：`go run ./cmd/main.go`
5. 健康检查：GET `http://localhost:8080/health`
6. 聊天 SSE 示例：POST `http://localhost:8080/sendMessage`（需在 Header 指定 `Content-Type: application/json`，Body 示例：`{"message":"hello","history":[]}`）

## 配置
支持通过环境变量 `CONFIG_FILE` 指定配置文件路径，默认读取根目录下的 `config.yaml`。

## 后续接入提示
- 在 `HandleChatStream` 内的 TODO 注释处接入 JWT 校验、配额校验与扣减、历史记录。
- 在 `middlewares.AuthMiddleware` 实现实际的 JWT 鉴权。
- 替换 `internal/handler/placeholder.go` 中的占位 Handler 为真实逻辑。
