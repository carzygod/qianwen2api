package main

import (
	"net/http"
	"qianwen2api/internal"
)

func main() {
	// 加载配置
	internal.LoadConfig()

	// 初始化日志
	internal.InitLogger()

	internal.LogInfo("===========================================")
	internal.LogInfo("Qianwen2API Server Starting...")
	internal.LogInfo("===========================================")
	internal.LogInfo("Port: %s", internal.Cfg.Port)
	internal.LogInfo("Pool Size: %d", internal.Cfg.PoolSize)
	internal.LogInfo("Log Level: %s", internal.Cfg.LogLevel)
	internal.LogInfo("===========================================")

	// 初始化账号池
	if err := internal.InitPool(internal.Cfg.PoolSize); err != nil {
		internal.LogError("Failed to initialize pool: %v", err)
		return
	}

	// 注册路由
	http.HandleFunc("/v1/models", internal.HandleModels)
	http.HandleFunc("/v1/chat/completions", internal.HandleChatCompletions)

	// 启动服务器
	addr := ":" + internal.Cfg.Port
	internal.LogInfo("Server listening on %s", addr)
	internal.LogInfo("===========================================")

	if err := http.ListenAndServe(addr, nil); err != nil {
		internal.LogError("Server failed: %v", err)
	}
}
