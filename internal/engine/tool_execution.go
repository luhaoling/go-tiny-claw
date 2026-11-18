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
func (e *AgentEngine) executeToolCalls(ctx context.Context, calls []schema.ToolCall) []schema.Message {
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
				observations[idx] = e.executeToolCall(ctx, idx, toolCall)
			}(i, call)
		}
		wg.Wait()
		return observations
	}

	for i, call := range calls {
		fmt.Println("存在写操作，串行执行")
		observations[i] = e.executeToolCall(ctx, i, call)
	}
	return observations
}

func (e *AgentEngine) executeToolCall(ctx context.Context, idx int, call schema.ToolCall) schema.Message {
	log.Printf(" -> [Tool-%d] 执行工具: %s\n", idx, call.Name)
	result := e.registry.Execute(ctx, call)
	if result.IsError {
		log.Printf(" -> ❌ 工具执行报错: %s\n", result.Output)
	} else {
		log.Printf(" -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
	}
	return schema.Message{Role: schema.RoleUser, Content: result.Output, ToolCallID: call.ID}
}
