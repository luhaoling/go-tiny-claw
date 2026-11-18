package engine

import (
	"AgentHarnes/internal/schema"
	"AgentHarnes/internal/tools"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type executionTestRegistry struct {
	readOnly map[string]bool
	execute  func(schema.ToolCall) schema.ToolResult
}

func (r *executionTestRegistry) Register(_ tools.BaseTool)                  {}
func (r *executionTestRegistry) GetAvailableTools() []schema.ToolDefinition { return nil }
func (r *executionTestRegistry) IsReadOnly(name string) bool                { return r.readOnly[name] }
func (r *executionTestRegistry) Execute(_ context.Context, call schema.ToolCall) schema.ToolResult {
	return r.execute(call)
}

func TestExecuteToolCallsRunsAllReadOnlyCallsConcurrentlyAndPreservesOrder(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	r := &executionTestRegistry{readOnly: map[string]bool{"read": true}}
	r.execute = func(call schema.ToolCall) schema.ToolResult {
		started <- struct{}{}
		<-release
		return schema.ToolResult{ToolCallID: call.ID, Output: call.ID}
	}
	e := &AgentEngine{registry: r}
	calls := []schema.ToolCall{{ID: "first", Name: "read"}, {ID: "second", Name: "read"}}
	done := make(chan []schema.Message, 1)
	go func() { done <- e.executeToolCalls(context.Background(), calls) }()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("read-only calls did not start concurrently")
		}
	}
	close(release)
	observations := <-done
	if len(observations) != 2 || observations[0].ToolCallID != "first" || observations[1].ToolCallID != "second" {
		t.Fatalf("observations are not in request order: %#v", observations)
	}
}

func TestExecuteToolCallsRunsEntireMixedBatchSequentially(t *testing.T) {
	var active, maxActive atomic.Int32
	var mu sync.Mutex
	var order []string
	r := &executionTestRegistry{
		readOnly: map[string]bool{"read": true}, // write intentionally absent
		execute: func(call schema.ToolCall) schema.ToolResult {
			current := active.Add(1)
			defer active.Add(-1)
			for {
				old := maxActive.Load()
				if current <= old || maxActive.CompareAndSwap(old, current) {
					break
				}
			}
			mu.Lock()
			order = append(order, call.ID)
			mu.Unlock()
			return schema.ToolResult{ToolCallID: call.ID, Output: fmt.Sprint(call.ID)}
		},
	}
	e := &AgentEngine{registry: r}
	calls := []schema.ToolCall{{ID: "read-1", Name: "read"}, {ID: "write", Name: "write"}, {ID: "read-2", Name: "read"}}
	observations := e.executeToolCalls(context.Background(), calls)
	if got := maxActive.Load(); got != 1 {
		t.Fatalf("max concurrent calls = %d, want 1", got)
	}
	want := []string{"read-1", "write", "read-2"}
	for i := range want {
		if order[i] != want[i] || observations[i].ToolCallID != want[i] {
			t.Fatalf("call/result order mismatch: calls=%v observations=%#v", order, observations)
		}
	}
}

func TestExecuteToolCallsTreatsUnknownToolAsWriteCapable(t *testing.T) {
	var active, maxActive atomic.Int32
	r := &executionTestRegistry{readOnly: map[string]bool{"read": true}}
	r.execute = func(call schema.ToolCall) schema.ToolResult {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			old := maxActive.Load()
			if current <= old || maxActive.CompareAndSwap(old, current) {
				break
			}
		}
		return schema.ToolResult{ToolCallID: call.ID}
	}
	e := &AgentEngine{registry: r}
	e.executeToolCalls(context.Background(), []schema.ToolCall{{ID: "1", Name: "read"}, {ID: "2", Name: "unregistered"}})
	if got := maxActive.Load(); got != 1 {
		t.Fatalf("max concurrent calls = %d, want 1", got)
	}
}
