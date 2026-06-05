package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sevens/SemanticBI/internal/config"
)

// ClientManager 管理 MCP 客户端连接
type ClientManager struct {
	sessions []*mcp.ClientSession
}

// NewClientManager 创建 MCP 客户端管理器
func NewClientManager(ctx context.Context, cfg *config.MCPConfig) (*ClientManager, error) {
	manager := &ClientManager{}

	// 连接 STDIO 模式的 MCP 服务器
	for name, srv := range cfg.StdioServers {
		session, err := connectStdio(ctx, name, srv)
		if err != nil {
			return nil, fmt.Errorf("connect stdio mcp server %s: %w", name, err)
		}
		manager.sessions = append(manager.sessions, session)
	}

	// 连接 SSE 模式的 MCP 服务器
	for name, srv := range cfg.SSEServers {
		session, err := connectSSE(ctx, name, srv)
		if err != nil {
			return nil, fmt.Errorf("connect sse mcp server %s: %w", name, err)
		}
		manager.sessions = append(manager.sessions, session)
	}

	return manager, nil
}

func connectStdio(ctx context.Context, name string, cfg *config.MCPStdioConfig) (*mcp.ClientSession, error) {
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if cfg.Env != nil {
		envList := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			envList = append(envList, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envList
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "SemanticBI",
		Version: "1.0.0",
	}, nil)

	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	session, err := client.Connect(connCtx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect stdio mcp %s: %w", name, err)
	}

	return session, nil
}

func connectSSE(ctx context.Context, name string, cfg *config.MCPSSEConfig) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "SemanticBI",
		Version: "1.0.0",
	}, nil)

	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	session, err := client.Connect(connCtx, &mcp.SSEClientTransport{
		Endpoint: cfg.URL,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect sse mcp %s: %w", name, err)
	}

	return session, nil
}

// GetAllTools 从所有 MCP 服务器获取工具列表
func (m *ClientManager) GetAllTools(ctx context.Context) ([]tool.BaseTool, error) {
	var allTools []tool.BaseTool

	for _, session := range m.sessions {
		tools, err := officialmcp.GetTools(ctx, &officialmcp.Config{
			Cli: session,
		})
		if err != nil {
			return nil, fmt.Errorf("get mcp tools: %w", err)
		}
		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// Close 关闭所有 MCP 客户端连接
func (m *ClientManager) Close() {
	for _, session := range m.sessions {
		_ = session.Close()
	}
}
