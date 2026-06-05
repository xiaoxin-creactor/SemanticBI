package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/sevens/SemanticBI/internal/agent"
	"github.com/sevens/SemanticBI/internal/config"
	"github.com/sevens/SemanticBI/internal/handler"
	"github.com/sevens/SemanticBI/internal/mcp"
	localTool "github.com/sevens/SemanticBI/internal/tool"
)

func main() {
	ctx := context.Background()

	// 0. 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("warn: load .env file: %v", err)
		}
	}

	// 1. 加载配置
	cfg := config.Load()
	if cfg.LLM.APIKey == "" {
		log.Fatal("LLM_API_KEY is required")
	}

	// 2. 初始化 MCP 连接（可选）
	mcpManager, err := mcp.NewClientManager(ctx, &cfg.MCP)
	if err != nil {
		log.Fatalf("init mcp: %v", err)
	}
	defer mcpManager.Close()

	// 3. 获取 MCP 工具
	mcpTools, err := mcpManager.GetAllTools(ctx)
	if err != nil {
		log.Printf("warn: get mcp tools: %v (MCP tools disabled)", err)
		mcpTools = nil
	}
	log.Printf("loaded %d MCP tools", len(mcpTools))

	// 4. 创建本地工具
	localTools := []tool.BaseTool{
		localTool.NewCalcTool(),
		localTool.NewTimeTool(),
	}

	// 5. 创建 ReAct Agent
	rAgent, err := agent.NewReActAgent(ctx, &cfg.LLM, localTools, mcpTools,
		`你是 SemanticBI 的智能助手。你可以使用各种工具来帮助用户解决问题。

## 可用工具
- calculator: 计算数学表达式
- get_current_time: 获取当前时间
- MCP 工具: 由连接的 MCP 服务器提供

请使用简体中文回答用户的问题。如果工具调用失败，请向用户说明原因。`,
	)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	// 6. 启动 HTTP 服务
	r := gin.Default()
	h := handler.NewHandler(rAgent)
	handler.SetupRouter(r, h)

	// 提供聊天页面
	r.StaticFile("/chat", "./static/chat.html")

	addr := ":" + cfg.Server.Port
	log.Printf("server starting on %s", addr)
	log.Printf("API: POST http://localhost%s/api/chat", addr)
	log.Printf("Health: GET http://localhost%s/ping", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
