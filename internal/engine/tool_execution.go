package engine

import (
	"AgentHarnes/internal/schema"
	"context"
	"fmt"
	"log"
	"sync"
)

// executeToolCalls runs a whole read-only batch concurrently. If any call is
// write-capable (including an unknown tool), the entire batch runs in order.
func (e *AgentEngine) executeToolCalls(ctx context.Context, calls []schema.ToolCall, reporter Reporter) []schema.Message {
	observations := make([]schema.Message, len(calls))
	allReadOnly := true
	for _, call := range calls {
		if !e.registry.IsReadOnly(call.Name) {
			allReadOnly = false
			break
		}
	}

	if allReadOnly {
		fmt.Println("全部是只读操作，并发加速")
		var wg sync.WaitGroup
		for i, call := range calls {
			wg.Add(1)
			go func(idx int, toolCall schema.ToolCall) {
				defer wg.Done()
				if reporter != nil {
					// 报告即将在底层执行的工具
					reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
				}
				observations[idx] = e.executeToolCall(ctx, idx, toolCall, reporter)
			}(i, call)
		}
		wg.Wait()

		return observations
	}

	for i, call := range calls {
		fmt.Println("存在写操作，串行执行")
		observations[i] = e.executeToolCall(ctx, i, call, reporter)
	}
	return observations
}

func (e *AgentEngine) executeToolCall(ctx context.Context, idx int, call schema.ToolCall, reporter Reporter) schema.Message {
	log.Printf(" -> [Tool-%d] 执行工具: %s\n", idx, call.Name)
	result := e.registry.Execute(ctx, call)
	if reporter != nil {
		// 为了防止大文件读取导致飞书消息过长被截断，仅汇报工具执行状态
		// 传递给大模型的 observationMsgs 依然是完整数据，人类看到的 Reporter 时缩略版的
		displayOutput := result.Output
		if len(displayOutput) > 200 {
			displayOutput = displayOutput[:00] + "...(已截断)"
		}
		reporter.OnToolResult(ctx, call.Name, displayOutput, result.IsError)
	}
	return schema.Message{Role: schema.RoleUser, Content: result.Output, ToolCallID: call.ID}
}
