package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ===== Calculator Tool =====

// CalculatorInput 计算器工具输入
type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"description=数学表达式，如 1+2, 3*4, 10/2 等"`
}

// CalculatorOutput 计算器工具输出
type CalculatorOutput struct {
	Result string `json:"result"`
}

// NewCalcTool 创建计算器工具
func NewCalcTool() tool.InvokableTool {
	return &calcTool{}
}

type calcTool struct{}

func (c *calcTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculator",
		Desc: "计算数学表达式，支持加减乘除运算",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"expression": {
				Type:     "string",
				Desc:     "数学表达式，如 1+2, 3*4, 10/2 等",
				Required: true,
			},
		}),
	}, nil
}

func (c *calcTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input CalculatorInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("parse arguments failed: %w", err)
	}

	// 简单的表达式计算（实际项目建议使用 expr 库）
	result, err := evalExpression(input.Expression)
	if err != nil {
		return "", err
	}

	out := CalculatorOutput{Result: result}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func evalExpression(expr string) (string, error) {
	// 这里使用简单的方法，实际项目可用 expr 库或 govaluate
	var a, b float64
	var op byte
	if n, _ := fmt.Sscanf(expr, "%f%c%f", &a, &op, &b); n == 3 {
		switch op {
		case '+':
			return fmt.Sprintf("%.2f", a+b), nil
		case '-':
			return fmt.Sprintf("%.2f", a-b), nil
		case '*':
			return fmt.Sprintf("%.2f", a*b), nil
		case '/':
			if b == 0 {
				return "", fmt.Errorf("division by zero")
			}
			return fmt.Sprintf("%.2f", a/b), nil
		}
	}

	return "", fmt.Errorf("unsupported expression: %s", expr)
}

// ===== Time Tool =====

// TimeInput 时间工具输入
type TimeInput struct {
	Format string `json:"format,omitempty" jsonschema:"description=时间格式，如 RFC3339, DateTime, DateOnly 等，默认为 DateTime"`
}

// NewTimeTool 创建时间工具
func NewTimeTool() tool.InvokableTool {
	return &timeTool{}
}

type timeTool struct{}

func (t *timeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_current_time",
		Desc: "获取当前日期和时间",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"format": {
				Type: "string",
				Desc: "时间格式: RFC3339, DateTime, DateOnly, TimeOnly. 默认为 DateTime",
			},
		}),
	}, nil
}

func (t *timeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input TimeInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &input)

	now := time.Now()
	var result string
	switch input.Format {
	case "RFC3339":
		result = now.Format(time.RFC3339)
	case "DateOnly":
		result = now.Format(time.DateOnly)
	case "TimeOnly":
		result = now.Format(time.TimeOnly)
	default:
		result = now.Format("2006-01-02 15:04:05")
	}

	b, _ := json.Marshal(map[string]string{"current_time": result})
	return string(b), nil
}

// 确保实现了接口
var (
	_ tool.InvokableTool = (*calcTool)(nil)
	_ tool.InvokableTool = (*timeTool)(nil)
)
