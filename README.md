# QianWen2API

千问 API 代理服务，将千问网页版 API 转换为 OpenAI 兼容格式。


## 特性

- OpenAI 兼容
- 支持流式和非流式响应
- 内置游客账号池管理
- 支持多轮对话
- 支持推理模型（reasoning_content）
- 禁用联网搜索

## 快速开始

### 克隆项目

```bash
git clone https://github.com/kao0312/qianwen2api.git
cd qianwen2api
```

### Docker 部署

```bash
# 本地构建
docker build -t qianwen2api:latest .

docker run -d -p 8080:8080 --name qianwen2api qianwen2api:latest

# （可选）自定义配置
docker run -d \
  -p 8080:8080 \
  -e POOL_SIZE=20 \
  -e AUTH_KEY=your-secret-key \
  -e LOG_LEVEL=info \
  -e REFRESH_HOURS=10 \
  --name qianwen2api \
  qianwen2api:latest
```

### 源码运行

```bash
go run main.go
```

## 配置说明

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `PORT` | 服务监听端口 | `8080` |
| `POOL_SIZE` | 游客账号池大小 | `20` |
| `AUTH_KEY` | API 鉴权密钥 **(不设置则不鉴权)** | 无 |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |
| `REFRESH_HOURS` | UMID Token 刷新周期 (小时) | `10` |

## API 使用

### 聊天接口

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tongyi-qwen3-max-model",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant"},
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true
  }'
```

### 模型列表

```bash
curl http://localhost:8080/v1/models
```

 - `tongyi-qwen3-max-model`
 - `tongyi-qwen3-max-thinking`

## 注意事项

1. **UMID Token 生成**：基于浏览器，首次启动需要较长时间
2. **刷新周期**：定时刷新 UMID Token，业务 token 在 20 轮对话后自动获取新的
