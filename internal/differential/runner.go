package differential

import (
	"context"
	"fmt"
)

const traceSemantics = "top-level pre-op PC/opcode/gas/stack; post-op gas and non-terminal stack derived at the next top-level opcode; terminal stack and memory are not compared"

type Engine struct {
	echo Runner
	geth Runner
}

func NewEngine(echo, geth Runner) *Engine { return &Engine{echo: echo, geth: geth} }

func DefaultEngine() *Engine { return NewEngine(EchoRunner{}, GethRunner{}) }

func (e *Engine) Compare(ctx context.Context, req Request) (ComparisonResult, error) {
	if e == nil || e.echo == nil || e.geth == nil {
		return ComparisonResult{}, fmt.Errorf("differential engine requires both runners")
	}
	normalized, err := normalizeRequest(req)
	if err != nil {
		return ComparisonResult{}, err
	}
	echo, err := e.echo.Run(ctx, normalized)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("EchoEVM runner: %w", err)
	}
	geth, err := e.geth.Run(ctx, normalized)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("geth runner: %w", err)
	}
	result := CompareResults(normalized, echo, geth)
	result.TraceSemantics = traceSemantics
	return result, nil
}
