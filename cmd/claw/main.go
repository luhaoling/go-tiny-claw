package main

import (
	"AgentHarnes/internal/engine"
	"AgentHarnes/internal/provider"
	"AgentHarnes/internal/schema"
	"context"
	"log"
	"os"
)

type mockProvider struct {
	turn int
}

func (m *mockProvider) Generate(ctx context.Context, msgs []schema.Message, tools []schema.ToolDefinition) (*schema.Message, error) {
	if len(tools) == 0 {
		return &schema.Message{
			Role:    schema.RoleAssistant,
			Content: "【推理中】目标是检查文件。我不能直接盲猜，我需要先调用 bash 工具执行 ls 命令，看看当前目录下有什么，然后再做定夺。",
		}, nil
	}
	m.turn++
	if m.turn == 1 {
		return &schema.Message{
			Role:    schema.RoleAssistant,
			Content: "我要执行刚才计划的步骤了。",
			ToolCalls: []schema.ToolCall{
				{
					ID:        "call_123",
					Name:      "bash",
					Arguments: []byte(`{"command":"ls -la"}`),
				},
			},
		}, nil
	}
	return &schema.Message{
		Role:    schema.RoleAssistant,
		Content: "根据工具返回结果，我看到了 main.go,任务圆满完成！",
	}, nil
}

type mockRegistry struct{}

func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition {
	return []schema.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的当前天气情况。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	log.Printf("  -> [Mock 工具执行] 获取 %s 的天气中...\n", call.Name)
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "API 返回:今天是晴天，气温 25 度。",
		IsError:    false,
	}
}
func main() {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatalf("请先设置 ZHIPU_API_KEY 环境变量")
	}
	workdir, _ := os.Getwd()

	// 1. 初始化真实的 Provider 大脑(指向智谱）
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	// 2. 注入伪造的工具注册表
	registry := &mockRegistry{}

	// 3. 实例化并运行引擎，开启 EnableThinking =true 开启慢思考模式
	eng := engine.NewAgentEngine(llmProvider, registry, workdir, false)

	// 4. 设定测试任务
	prompt := "我想去北京跑步，帮我查查天气合适吗？"
	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎崩溃:%v", err)
	}

}

//
//func main() {
//	fmt.Println("🚀 欢迎来到 go-tiny-claw 引擎启动序列")
//
//	// todo: 1. 初始化模型 Provider（大脑）
//	// provider:=provider.NewClaudeProvider(...)
//
//	// todo: 2. 初始化 Tool Registry（手脚）
//	// registry：=tools.NewRegistry()
//	// registry.Register(tools.NewBashTool())
//
//	// todo: 3. 初始化上下文管理器（内存管理器）
//	// ctxManager:=context.NewManager(...)
//
//	// todo: 4. 组装并启动核心 Engine（操作系统心脏）
//	// engine：=engine.NewAgentEngine(provider,ctxManger)
//
//	// fmt.Println("开始执行任务...")
//	// err:=engine.Run("帮我检查一下当前目录下的文件并输出一个 README.md 大纲")
//	// if err!=nil{
//	//	 log.Fatalf("引擎运行崩溃:%v",err)
//	// }
//
//	log.Println("架构蓝图搭建完毕，等待各核心模块注入")
//
//}
