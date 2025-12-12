## 项目简介
基于 Gin 的 LLM 对话后端骨架，包含配置加载、SSE 流式聊天接口、语音 STT/TTS 占位实现、鉴权占位与路由分组。

## 目录结构
- `cmd/main.go`：入口，加载配置并启动服务
- `config.yaml`：示例配置（服务端口、调试开关、LLM 地址与密钥）
- `internal/config`：配置结构体与 YAML 加载
- `internal/router`：路由注册，登录/聊天/语音/管理模块及鉴权中间件占位
- `internal/handler`：
  - `chat_handler.go`：聊天 SSE 转发逻辑，含 LLM 上游调用占位与配额/JWT TODO
  - `stt_handler.go`：STT 上传占位，实现文件接收与云 STT 对接 TODO
  - `tts_handler.go`：TTS 转换占位，模拟生成 MP3 字节流
  - `placeholder.go`：其余接口占位返回
- `internal/middlewares/auth.go`：鉴权中间件占位

## 快速开始
1. 安装 Go 1.21+
2. 拉取依赖：`go mod tidy`
3. 运行：`go run ./cmd/main.go`
4. 健康检查：`curl http://localhost:8080/health`
5. SSE 聊天示例：
   ```
   curl -N -H "Content-Type: application/json" \
     -X POST http://localhost:8080/sendMessage \
     -d '{"message":"hello","history":[]}'
   ```
6. STT 上传示例（返回占位识别文本）：`curl -X POST http://localhost:8080/upload -F "audio_file=@/path/to/audio.wav"`
7. TTS 转换示例（下载占位音频）：`curl -X POST http://localhost:8080/convert -H "Content-Type: application/json" -d '{"text":"你好"}' -o tts_output.mp3`

## 配置
- 默认读取根目录 `config.yaml`，可用环境变量 `CONFIG_FILE` 覆盖。
- 示例字段：
  - `server.addr`: 监听地址，默认 `:8080`
  - `server.debug`: 是否启用 Gin Debug 模式
  - `llm.base_url`: LLM 上游地址（流式接口）
  - `llm.api_key`: LLM 访问密钥

## 已注册接口
- 无鉴权：`POST /login`，`POST /setPassword`，`POST /refreshToken`
- 聊天（鉴权占位）：`POST /sendMessage`(SSE)，`GET /getChatHistory`，`POST /newChat`，`PUT /renameChat`，`DELETE /deleteChat`，`GET /getQuota`
- STT/TTS（鉴权占位）：`POST /upload`，`POST /convert`
- 管理（鉴权占位）：`POST /addUser`，`DELETE /deleteUser`，`POST /setQuota`，`GET /getUser`

## 后续接入提示
- 在 `HandleChatStream` 的 TODO 处接入 JWT 校验、配额校验与扣减、历史存储，并对接真实 LLM 流接口。
- 在 `HandleSTTUpload` / `HandleTTSConvert` 的 TODO 处对接云 STT/TTS 服务。
- 在 `middlewares.AuthMiddleware` 完成 JWT/权限校验。
- 替换 `placeholder.go` 中的占位 Handler 为真实业务逻辑。
