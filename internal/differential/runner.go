package differential

import (
	"context"
	"fmt"

	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

const traceSemantics = "top-level pre-op PC/opcode/gas/stack; post-op gas and non-terminal stack derived at the next top-level opcode; terminal stack and memory are not compared"

type Engine struct {
	echo      Runner
	reference Runner
}

func NewEngine(echo, reference Runner) *Engine { return &Engine{echo: echo, reference: reference} }

func DefaultEngine() *Engine { return NewEngine(EchoRunner{}, nil) }

// RunEcho executes a request with EchoEVM only while preserving the same
// validation and normalization contract used by Compare.
func (e *Engine) RunEcho(ctx context.Context, req Request) (ExecutionResult, error) {
	if e == nil || e.echo == nil {
		return ExecutionResult{}, fmt.Errorf("differential engine requires an EchoEVM runner")
	}
	normalized, err := normalizeRequest(req)
	if err != nil {
		return ExecutionResult{}, err
	}
	result, err := e.echo.Run(ctx, normalized)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("EchoEVM runner: %w", err)
	}
	return result, nil
}

// RunEchoExplain preserves RunEcho validation while returning nested,
// explainable opcode events from runners that expose that capability.
func (e *Engine) RunEchoExplain(ctx context.Context, req Request, maxMemoryBytes int) (ExecutionResult, []explaintrace.OpcodeEvent, error) {
	if e == nil || e.echo == nil {
		return ExecutionResult{}, nil, fmt.Errorf("differential engine requires an EchoEVM runner")
	}
	runner, ok := e.echo.(interface {
		RunExplain(context.Context, Request, int) (ExecutionResult, []explaintrace.OpcodeEvent, error)
	})
	if !ok {
		return ExecutionResult{}, nil, fmt.Errorf("EchoEVM runner does not support explainable traces")
	}
	normalized, err := normalizeRequest(req)
	if err != nil {
		return ExecutionResult{}, nil, err
	}
	result, events, err := runner.RunExplain(ctx, normalized, maxMemoryBytes)
	if err != nil {
		return ExecutionResult{}, nil, fmt.Errorf("EchoEVM runner: %w", err)
	}
	return result, events, nil
}

func (e *Engine) Compare(ctx context.Context, req Request) (ComparisonResult, error) {
	if e == nil || e.echo == nil || e.reference == nil {
		return ComparisonResult{}, fmt.Errorf("comparison requires an explicit independent reference runner")
	}
	normalized, err := normalizeRequest(req)
	if err != nil {
		return ComparisonResult{}, err
	}
	echo, err := e.echo.Run(ctx, normalized)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("EchoEVM runner: %w", err)
	}
	reference, err := e.reference.Run(ctx, normalized)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("reference runner: %w", err)
	}
	result := CompareResults(normalized, echo, reference)
	result.TraceSemantics = traceSemantics
	return result, nil
}
