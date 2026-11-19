package engine

import (
	"AgentHarnes/internal/provider"
	"AgentHarnes/internal/schema"
	"AgentHarnes/internal/tools"
	"context"
	"fmt"
	"log"
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
func (e *AgentEngine) Run(ctx context.Context, userPrompt string, reporter Reporter) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)

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

		// 获取当前挂载的所有工具定义
		availableTools := e.registry.GetAvailableTools()

		// ======================================================================
		// Phase 1: 慢思考阶段(Thinking) - 剥夺工具，强制规划
		// ======================================================================
		if e.EnableThinking {
			if reporter != nil {
				// 触发 Reporter：开始慢思考
				reporter.OnThinking(ctx)
			}
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
		if responseMsg.Content != "" && reporter != nil {
			// 触发 Reporter：输出阶段性总结或最终回复
			reporter.OnMessage(ctx, responseMsg.Content)
		}

		// 3.退出条件判断
		// 如果模型没有请求任何工具调用，说明它认为任务已经完成，跳出循环
		if len(responseMsg.ToolCalls) == 0 {
			log.Println("[Engine] 任务完成，退出循环。")
			break
		}

		// 4. 执行工具批次并收集观察结果
		observationMsgs := e.executeToolCalls(ctx, responseMsg.ToolCalls, reporter)
		log.Printf("[Engine] 本批次工具执行完毕，开始聚合观察结果（Observation）...")
		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
	}
	return nil
}
