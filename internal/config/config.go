package config

import (
	"os"
	"strings"
)

// LLMConfig 大模型配置
type LLMConfig struct {
	APIKey  string // API 密钥
	BaseURL string // API 基础地址
	Model   string // 模型名称
}

// MCPStdioConfig STDIO 模式 MCP 服务器配置
type MCPStdioConfig struct {
	Command string            // 可执行命令
	Args    []string          // 命令行参数
	Env     map[string]string // 环境变量
}

// MCPSSEConfig SSE 模式 MCP 服务器配置
type MCPSSEConfig struct {
	URL     string
	Headers map[string]string
}

// MCPConfig MCP 服务器配置
type MCPConfig struct {
	StdioServers map[string]*MCPStdioConfig // name -> stdio config
	SSEServers   map[string]*MCPSSEConfig   // name -> sse config
}

// Config 全局配置
type Config struct {
	LLM    LLMConfig
	MCP    MCPConfig
	Server ServerConfig
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port string
}

// Load 从环境变量加载配置
func Load() *Config {
	cfg := &Config{
		LLM: LLMConfig{
			APIKey:  os.Getenv("LLM_API_KEY"),
			BaseURL: os.Getenv("LLM_BASE_URL"),
			Model:   os.Getenv("LLM_MODEL"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		MCP: MCPConfig{
			StdioServers: make(map[string]*MCPStdioConfig),
			SSEServers:   make(map[string]*MCPSSEConfig),
		},
	}

	// 解析 MCP stdio 服务器配置 (MCP_SERVER__{NAME}__COMMAND / ARGS)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "MCP_SERVER__") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		value := parts[1]

		// 格式: MCP_SERVER__{NAME}__COMMAND 或 MCP_SERVER__{NAME}__ARGS
		keyParts := strings.Split(key, "__")
		if len(keyParts) < 3 {
			continue
		}
		name := keyParts[1]
		field := keyParts[2]

		switch field {
		case "COMMAND":
			if cfg.MCP.StdioServers[name] == nil {
				cfg.MCP.StdioServers[name] = &MCPStdioConfig{
					Args: []string{},
					Env:  make(map[string]string),
				}
			}
			cfg.MCP.StdioServers[name].Command = value
		case "ARGS":
			if cfg.MCP.StdioServers[name] == nil {
				cfg.MCP.StdioServers[name] = &MCPStdioConfig{}
			}
			cfg.MCP.StdioServers[name].Args = strings.Split(value, ",")
		case "URL":
			if cfg.MCP.SSEServers[name] == nil {
				cfg.MCP.SSEServers[name] = &MCPSSEConfig{
					Headers: make(map[string]string),
				}
			}
			cfg.MCP.SSEServers[name].URL = value
		}
	}

	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o"
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
