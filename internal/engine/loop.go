package engine

import (
	"AgentHarnes/internal/provider"
	"AgentHarnes/internal/schema"
	"AgentHarnes/internal/tools"
	"context"
	"fmt"
	"log"
	"sync"
)

// AgentEngine 微型 OS 的核心驱动
type AgentEngine struct {
	provider provider.LLMProvider
	registry tools.Registry

	// workDir (工作区)：借鉴 OpenClaw 的理念，Agent 必须有一个明确的物理边界
	WorkDir        string
	EnableThinking bool // 慢思考开关
}

func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// Run 启动 Agent 的生命周期
func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式(Thinking Phase):%v\n", e.EnableThinking)

	// 1. 初始化会话的 Context （上下文内存）
	// 在真实的场景中，这里会由动态 Prompt 组装器加载 AGENT.md。
	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}
	turnCount := 0

	// 2. The Main Loop：心跳开始（标准的 ReAct 循环）
	for {
		turnCount++
		log.Printf("============ [Turn %d] 开始 ============\n", turnCount)

		// 获取当前挂载的所有工具定义
		availableTools := e.registry.GetAvailableTools()

		// ======================================================================
		// Phase 1: 慢思考阶段(Thinking) - 剥夺工具，强制规划
		// ======================================================================
		if e.EnableThinking {
			log.Printf("[Engine] [Phase 1] 剥夺工具访问权，强制进入慢思考与规划阶段...")

			// 核心机制：传入 availableTools 为 nil!
			// 大模型看不到任何 JSON Schema, 被迫只能输出纯文本的思考过程。
			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("thinking 阶段生成失败:%w", err)
			}
			if thinkResp.Content != "" {
				fmt.Printf("🧠 [内部思考 Trace]: %s\n", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}
		}

		// ======================================================================
		// Phase 2: 行动阶段 (Action) - 恢复工具，顺着规划执行
		// ======================================================================
		log.Println("[Engine][Phase 2] 恢复工具挂载，等待模型采取行动...")
		// 此时的模型 contextHistory 已经包含上一轮模型自己的 Thinking Trace
		// 模型会顺着自己的逻辑，结合恢复的 availableTools 发起精准的工具调用。

		log.Println("[Engine] 正在思考(reasoning)...")
		responseMsg, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("模型生成失败:%w", err)
		}

		// 将模型的响应完整地追加到上下文历史中
		contextHistory = append(contextHistory, *responseMsg)

		// 如果模型回复了纯文本，打印出来（这通常是它的思考过程或者是最终结果）
		if responseMsg.Content != "" {
			fmt.Printf("🤖 模型: %s \n", responseMsg.Content)
		}

		// 3.退出条件判断
		// 如果模型没有请求任何工具调用，说明它认为任务已经完成，跳出循环
		if len(responseMsg.ToolCalls) == 0 {
			log.Println("[Engine] 任务完成，退出循环。")
			break
		}

		// 4. 执行行动（Action）与 观察结果（Observation）
		log.Printf("[Engine] 模型请求并发调用 %d 个工具...\n", len(responseMsg.ToolCalls))

		// 5. 预分配一个固定长度的切片，用于安全地存放各个并发工具的执行结果（Observation
		observationMsgs := make([]schema.Message, len(responseMsg.ToolCalls))

		// 6. 声明 WaitGroup 用于阻塞等待所有协程完成
		var wg sync.WaitGroup

		//7. 循环遍历模型请求所有工具，为每一个工具单独 fork 一个 groutine
		for i, toolCall := range responseMsg.ToolCalls {
			wg.Add(1) // 增加计数器

			// 开启协程
			go func(idx int, call schema.ToolCall) {
				defer wg.Done()
				log.Printf(" -> [Go-%d] 🛠️ 触发并行执行:%s\n", idx, call.Name)
				// 调用底层 Registry 执行工具(物理操作)
				result := e.registry.Execute(ctx, call)

				if result.IsError {
					log.Printf(" -> ❌ 工具执行报错: %s\n", result.Output)
				} else {
					log.Printf(" -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
				}

				// 将工具执行的观察结果（Observation）封装为 User Message 追加到上下文中
				// 注意 ToolCallID 必须携带！这是维系大模型推理链条的关键
				observationMsg := schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: toolCall.ID,
				}
				observationMsgs[idx] = observationMsg
			}(i, toolCall)
		}
		// 7. 阻塞等待：主循环挂起，直到所有的并发携程全部执行完毕
		wg.Wait()
		log.Printf("[Engine] 所有并发工具执行完毕，开始聚合观察结果（Observation）...")
		// 8. 聚合装填：将并行的结果，按照原本的顺序，一次性追加到上下文时间线中
		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
	}
	return nil
}
