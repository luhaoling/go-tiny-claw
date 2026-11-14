package main

import (
	"AgentHarnes/internal/engine"
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
	return []schema.ToolDefinition{{Name: "bash"}}
}

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "-rw-r--r-- 1 user group 234 Oct 24 10:00 main.go\n",
		IsError:    false,
	}
}
func main() {
	workdir, _ := os.Getwd()
	p := &mockProvider{}
	r := &mockRegistry{}

	eng := engine.NewAgentEngine(p, r, workdir, false)
	err := eng.Run(context.Background(), "帮我检查当前目录的文件")
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
