package differential

import (
	"fmt"
	"reflect"
)

func CompareResults(req Request, echo, geth ExecutionResult) ComparisonResult {
	result := ComparisonResult{
		StatusMatch:     echo.Status == geth.Status,
		ReturnDataMatch: echo.ReturnData == geth.ReturnData,
		GasMatch:        echo.GasUsed == geth.GasUsed,
		StorageMatch:    reflect.DeepEqual(echo.Storage, geth.Storage),
		EchoEVM:         echo, Geth: geth, Request: req,
	}
	result.TraceMatch, result.FirstDivergence = compareTrace(echo.Trace, geth.Trace)
	if result.FirstDivergence == nil {
		switch {
		case !result.StatusMatch:
			result.FirstDivergence = resultDivergence("result", "status", echo.Status, geth.Status)
		case !result.ReturnDataMatch:
			result.FirstDivergence = resultDivergence("result", "returnData", echo.ReturnData, geth.ReturnData)
		case !result.GasMatch:
			result.FirstDivergence = resultDivergence("result", "gasUsed", echo.GasUsed, geth.GasUsed)
		case !result.StorageMatch:
			result.FirstDivergence = resultDivergence("result", "storage", echo.Storage, geth.Storage)
		}
	}
	result.Match = result.StatusMatch && result.ReturnDataMatch && result.GasMatch && result.StorageMatch && result.TraceMatch
	return result
}

func compareTrace(echo, geth []NormalizedStep) (bool, *Divergence) {
	limit := len(echo)
	if len(geth) < limit {
		limit = len(geth)
	}
	for i := 0; i < limit; i++ {
		a, b := echo[i], geth[i]
		checks := []struct {
			field string
			a, b  any
		}{
			{"pc", a.PC, b.PC}, {"opcode", a.Opcode, b.Opcode}, {"opcodeName", a.OpcodeName, b.OpcodeName},
			{"gasBefore", a.GasBefore, b.GasBefore}, {"gasAfter", a.GasAfter, b.GasAfter},
			{"stackBefore", a.StackBefore, b.StackBefore}, {"stackAfter", a.StackAfter, b.StackAfter},
			{"haltClass", a.HaltClass, b.HaltClass},
		}
		for _, check := range checks {
			if !reflect.DeepEqual(check.a, check.b) {
				step, pc := i, a.PC
				return false, &Divergence{Kind: "trace", Step: &step, PC: &pc, Opcode: a.OpcodeName,
					Field: check.field, EchoEVM: check.a, Geth: check.b,
					Description: fmt.Sprintf("trace step %d differs at %s", i, check.field)}
			}
		}
	}
	if len(echo) != len(geth) {
		step := limit
		var pc *uint64
		opcode := ""
		if limit < len(echo) {
			value := echo[limit].PC
			pc = &value
			opcode = echo[limit].OpcodeName
		}
		if limit < len(geth) && pc == nil {
			value := geth[limit].PC
			pc = &value
			opcode = geth[limit].OpcodeName
		}
		return false, &Divergence{Kind: "trace", Step: &step, PC: pc, Opcode: opcode,
			Field: "length", EchoEVM: len(echo), Geth: len(geth), Description: "normalized trace lengths differ"}
	}
	return true, nil
}

func resultDivergence(kind, field string, echo, geth any) *Divergence {
	return &Divergence{Kind: kind, Field: field, EchoEVM: echo, Geth: geth, Description: fmt.Sprintf("execution results differ at %s", field)}
}
