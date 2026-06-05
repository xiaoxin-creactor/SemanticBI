package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// ChatRequest 聊天请求
type ChatRequest struct {
	Message    string `json:"message" binding:"required"`
	Stream     bool   `json:"stream"`                // 是否使用流式响应
	SystemHint string `json:"system_hint,omitempty"` // 可选的系统提示补充
}

// Handler HTTP 请求处理器
type Handler struct {
	agent *react.Agent
}

// NewHandler 创建 Handler
func NewHandler(rAgent *react.Agent) *Handler {
	return &Handler{agent: rAgent}
}

// Chat 处理聊天请求（流式 SSE）
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构建消息
	messages := []*schema.Message{
		{Role: schema.User, Content: req.Message},
	}

	if req.Stream {
		h.handleStream(c, messages)
	} else {
		h.handleNonStream(c, messages)
	}
}

func writeSSE(w gin.ResponseWriter, flusher http.Flusher, v any) {
	data, _ := json.Marshal(v)
	_, _ = w.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
	flusher.Flush()
}

func (h *Handler) handleStream(c *gin.Context, messages []*schema.Message) {
	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// 立即发送 start 事件，让前端知道连接已建立
	writeSSE(c.Writer, flusher, map[string]string{"type": "start"})

	// 启动心跳 goroutine，在 agent.Stream() 之前开始，确保连接活跃
	doneCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				writeSSE(c.Writer, flusher, map[string]string{"type": "ping"})
			case <-doneCh:
				return
			}
		}
	}()
	defer close(doneCh)

	sr, err := h.agent.Stream(c.Request.Context(), messages, agent.WithComposeOptions())
	if err != nil {
		writeSSE(c.Writer, flusher, map[string]string{"type": "error", "content": fmt.Sprintf("agent stream error: %v", err)})
		return
	}
	defer sr.Close()

	// 简单直接地循环读取流消息
	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				writeSSE(c.Writer, flusher, map[string]string{"type": "done"})
				return
			}
			writeSSE(c.Writer, flusher, map[string]string{"type": "error", "content": err.Error()})
			return
		}

		// 发送文本内容
		if msg.Content != "" {
			writeSSE(c.Writer, flusher, map[string]string{"type": "token", "content": msg.Content})
		}

		// 发送工具调用事件
		for _, tc := range msg.ToolCalls {
			writeSSE(c.Writer, flusher, map[string]any{
				"type":      "tool_call",
				"id":        tc.ID,
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			})
		}

		// 检查客户端是否断开
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
	}
}

func (h *Handler) handleNonStream(c *gin.Context, messages []*schema.Message) {
	msg, err := h.agent.Generate(c.Request.Context(), messages, agent.WithComposeOptions())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": msg.Content,
		"role":    msg.Role,
	})
}

// Ping 健康检查
func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SetupRouter 配置路由
func SetupRouter(r *gin.Engine, h *Handler) {
	r.GET("/ping", h.Ping)
	r.POST("/api/chat", h.Chat)
}
