package agent

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/sevens/SemanticBI/internal/config"
)

// NewReActAgent 创建一个 ReAct Agent
// chatModel: 大语言模型
// localTools: 本地自定义工具列表
// mcpTools: 从 MCP 服务器获取的工具列表
// systemPrompt: 系统提示词
func NewReActAgent(
	ctx context.Context,
	cfg *config.LLMConfig,
	localTools []tool.BaseTool,
	mcpTools []tool.BaseTool,
	systemPrompt string,
) (*react.Agent, error) {
	// 1. 创建 OpenAI ChatModel
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	// 2. 合并所有工具
	allTools := make([]tool.BaseTool, 0, len(localTools)+len(mcpTools))
	allTools = append(allTools, localTools...)
	allTools = append(allTools, mcpTools...)

	// 3. 创建 ReAct Agent
	rAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: allTools,
		},
		MaxStep: 50,
		// DeepSeek 会先输出文字再输出 tool call，默认 checker 只看第一个 chunk
		// 所以需要一个读完整个流再判断的 checker
		StreamToolCallChecker: func(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
			defer sr.Close()
			for {
				msg, err := sr.Recv()
				if err == io.EOF {
					return false, nil
				}
				if err != nil {
					return false, err
				}
				if len(msg.ToolCalls) > 0 {
					return true, nil
				}
			}
		},
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			if systemPrompt != "" {
				return append([]*schema.Message{
					{Role: schema.System, Content: systemPrompt},
				}, input...)
			}
			return input
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent: %w", err)
	}

	return rAgent, nil
}
