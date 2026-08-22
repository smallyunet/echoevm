package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	ethabi "github.com/umbracle/ethgo/abi"
)

// buildCallData creates ABI encoded calldata from a function signature and
// comma-separated arguments.
//
// Supported types:
//   - uint8, uint16, ..., uint256 (decimal or 0x hex)
//   - int8, int16, ..., int256 (decimal or 0x hex)
//   - bool (true/false)
//   - string (UTF-8)
//   - address (0x-prefixed 40 hex chars)
//   - bytes (0x-prefixed dynamic hex)
//   - bytes1, bytes2, ..., bytes32 (0x-prefixed fixed hex)
//   - T[] arrays where T is a supported type (semicolon-separated values in brackets)
//
// Array syntax example: "[1;2;3]" for uint256[] or "[0xabc...;0xdef...]" for address[]
func buildCallData(sig, argString string) ([]byte, error) {
	open := strings.Index(sig, "(")
	close := strings.LastIndex(sig, ")")
	if open == -1 || close == -1 || close < open {
		return nil, fmt.Errorf("invalid function signature")
	}
	typesPart := sig[open+1 : close]
	typeNames := []string{}
	if len(typesPart) > 0 {
		typeNames = splitTypeNames(typesPart)
	}

	args := []string{}
	if len(argString) > 0 {
		args = splitArgs(argString)
	}

	if len(typeNames) != len(args) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d", len(typeNames), len(args))
	}

	tuple, err := ethabi.NewType("tuple(" + typesPart + ")")
	if err != nil {
		return nil, fmt.Errorf("invalid function signature: %w", err)
	}
	values := make([]interface{}, len(args))
	for i, element := range tuple.TupleElems() {
		val, err := parseArg(args[i], element.Elem)
		if err != nil {
			return nil, fmt.Errorf("argument %d (%s): %w", i, typeNames[i], err)
		}
		values[i] = val
	}
	encoded, err := tuple.Encode(values)
	if err != nil {
		return nil, err
	}
	selector := crypto.Keccak256([]byte(sig))[:4]
	return append(selector, encoded...), nil
}

// splitTypeNames splits type names handling nested arrays properly
func splitTypeNames(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, c := range s {
		switch c {
		case '[':
			depth++
			current.WriteRune(c)
		case ']':
			depth--
			current.WriteRune(c)
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(c)
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

// splitArgs splits arguments, respecting array brackets
func splitArgs(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, c := range s {
		switch c {
		case '[':
			depth++
			current.WriteRune(c)
		case ']':
			depth--
			current.WriteRune(c)
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(c)
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

func parseArg(val string, typ *ethabi.Type) (interface{}, error) {
	// Handle arrays
	if typ.Kind() == ethabi.KindSlice {
		return parseArrayArg(val, typ.Elem())
	}

	// Handle fixed-size arrays
	if typ.Kind() == ethabi.KindArray {
		return parseFixedArrayArg(val, typ.Elem(), typ.Size())
	}

	switch typ.Kind() {
	case ethabi.KindUInt:
		n, err := parseBigInt(val)
		if err != nil {
			return nil, err
		}
		// Convert to appropriate size
		return convertToUintType(n, typ.Size())

	case ethabi.KindInt:
		n, err := parseBigInt(val)
		if err != nil {
			return nil, err
		}
		// For int types, we keep as *big.Int
		return n, nil

	case ethabi.KindBool:
		return strings.ToLower(val) == "true", nil

	case ethabi.KindString:
		return val, nil

	case ethabi.KindAddress:
		if !strings.HasPrefix(val, "0x") {
			val = "0x" + val
		}
		if len(val) != 42 {
			return nil, fmt.Errorf("invalid address length: %s", val)
		}
		return common.HexToAddress(val), nil

	case ethabi.KindBytes:
		// Dynamic bytes
		return parseHexBytes(val)

	case ethabi.KindFixedBytes:
		// bytes1, bytes2, ..., bytes32
		b, err := parseHexBytes(val)
		if err != nil {
			return nil, err
		}
		return toFixedBytes(b, typ.Size())

	case ethabi.KindTuple:
		// Tuples are parsed from "(val1,val2,...)" syntax
		return parseTupleArg(val, typ)

	default:
		return nil, fmt.Errorf("unsupported type: %s", typ.String())
	}
}

// parseBigInt parses a decimal or hex string to *big.Int
func parseBigInt(val string) (*big.Int, error) {
	n := new(big.Int)
	var ok bool
	if strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X") {
		n, ok = n.SetString(val[2:], 16)
	} else if strings.HasPrefix(val, "-0x") || strings.HasPrefix(val, "-0X") {
		n, ok = n.SetString("-"+val[3:], 16)
	} else {
		n, ok = n.SetString(val, 10)
	}
	if !ok {
		return nil, fmt.Errorf("invalid integer value: %s", val)
	}
	return n, nil
}

// convertToUintType converts a big.Int to the appropriate uint type based on size
func convertToUintType(n *big.Int, bitSize int) (interface{}, error) {
	switch bitSize {
	case 8:
		return uint8(n.Uint64()), nil
	case 16:
		return uint16(n.Uint64()), nil
	case 32:
		return uint32(n.Uint64()), nil
	case 64:
		return n.Uint64(), nil
	default:
		// For uint128, uint256, etc., return *big.Int
		return n, nil
	}
}

// parseHexBytes parses a hex string (with or without 0x) to []byte
func parseHexBytes(val string) ([]byte, error) {
	val = strings.TrimPrefix(val, "0x")
	val = strings.TrimPrefix(val, "0X")
	if len(val)%2 != 0 {
		val = "0" + val
	}
	return hex.DecodeString(val)
}

// toFixedBytes converts a byte slice to a fixed-size byte array
func toFixedBytes(b []byte, size int) (interface{}, error) {
	if len(b) > size {
		return nil, fmt.Errorf("bytes too long: got %d, max %d", len(b), size)
	}

	// Pad with zeros on the right for fixed bytes
	padded := make([]byte, size)
	copy(padded, b)

	switch size {
	case 1:
		return [1]byte(padded), nil
	case 2:
		return [2]byte(padded), nil
	case 3:
		return [3]byte(padded), nil
	case 4:
		return [4]byte(padded), nil
	case 5:
		return [5]byte(padded), nil
	case 6:
		return [6]byte(padded), nil
	case 7:
		return [7]byte(padded), nil
	case 8:
		return [8]byte(padded), nil
	case 9:
		return [9]byte(padded), nil
	case 10:
		return [10]byte(padded), nil
	case 11:
		return [11]byte(padded), nil
	case 12:
		return [12]byte(padded), nil
	case 13:
		return [13]byte(padded), nil
	case 14:
		return [14]byte(padded), nil
	case 15:
		return [15]byte(padded), nil
	case 16:
		return [16]byte(padded), nil
	case 17:
		return [17]byte(padded), nil
	case 18:
		return [18]byte(padded), nil
	case 19:
		return [19]byte(padded), nil
	case 20:
		return [20]byte(padded), nil
	case 21:
		return [21]byte(padded), nil
	case 22:
		return [22]byte(padded), nil
	case 23:
		return [23]byte(padded), nil
	case 24:
		return [24]byte(padded), nil
	case 25:
		return [25]byte(padded), nil
	case 26:
		return [26]byte(padded), nil
	case 27:
		return [27]byte(padded), nil
	case 28:
		return [28]byte(padded), nil
	case 29:
		return [29]byte(padded), nil
	case 30:
		return [30]byte(padded), nil
	case 31:
		return [31]byte(padded), nil
	case 32:
		return [32]byte(padded), nil
	default:
		return nil, fmt.Errorf("unsupported fixed bytes size: %d", size)
	}
}

// parseArrayArg parses an array argument like "[1;2;3]" or "[0xabc;0xdef]"
func parseArrayArg(val string, elemType *ethabi.Type) (interface{}, error) {
	val = strings.TrimSpace(val)
	if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
		return nil, fmt.Errorf("array must be enclosed in brackets: %s", val)
	}

	inner := val[1 : len(val)-1]
	if inner == "" {
		// Empty array
		return makeEmptySlice(elemType)
	}

	// Split by semicolon, respecting nested brackets
	parts := splitArrayArgs(inner)
	elements := make([]interface{}, len(parts))

	for i, part := range parts {
		elem, err := parseArg(strings.TrimSpace(part), elemType)
		if err != nil {
			return nil, fmt.Errorf("array element %d: %w", i, err)
		}
		elements[i] = elem
	}

	return buildTypedSlice(elements, elemType)
}

func buildTypedArray(elements []interface{}, size int) (interface{}, error) {
	if len(elements) != size {
		return nil, fmt.Errorf("fixed array size mismatch")
	}
	if size == 0 {
		return nil, fmt.Errorf("zero-length fixed arrays are unsupported")
	}
	elementType := reflect.TypeOf(elements[0])
	array := reflect.New(reflect.ArrayOf(size, elementType)).Elem()
	for index, element := range elements {
		value := reflect.ValueOf(element)
		if value.Type() != elementType {
			return nil, fmt.Errorf("array element type mismatch")
		}
		array.Index(index).Set(value)
	}
	return array.Interface(), nil
}

// parseFixedArrayArg parses a fixed-size array
func parseFixedArrayArg(val string, elemType *ethabi.Type, size int) (interface{}, error) {
	val = strings.TrimSpace(val)
	if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
		return nil, fmt.Errorf("array must be enclosed in brackets: %s", val)
	}

	inner := val[1 : len(val)-1]
	// Split by semicolon, respecting nested brackets
	parts := splitArrayArgs(inner)

	if len(parts) != size {
		return nil, fmt.Errorf("fixed array size mismatch: expected %d, got %d", size, len(parts))
	}

	elements := make([]interface{}, size)
	for i, part := range parts {
		elem, err := parseArg(strings.TrimSpace(part), elemType)
		if err != nil {
			return nil, fmt.Errorf("array element %d: %w", i, err)
		}
		elements[i] = elem
	}

	return buildTypedArray(elements, size)
}

// splitArrayArgs splits array arguments by semicolon while respecting nested brackets
func splitArrayArgs(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, c := range s {
		switch c {
		case '[':
			depth++
			current.WriteRune(c)
		case ']':
			depth--
			current.WriteRune(c)
		case ';':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(c)
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

// makeEmptySlice creates an empty slice of the appropriate type
func makeEmptySlice(elemType *ethabi.Type) (interface{}, error) {
	switch elemType.Kind() {
	case ethabi.KindSlice, ethabi.KindArray:
		// Recursive call to get the element type's empty slice
		innerSlice, err := makeEmptySlice(elemType.Elem())
		if err != nil {
			return nil, err
		}
		// innerSlice is an instance of the element type (e.g. []*big.Int for nested uint256[])
		innerType := reflect.TypeOf(innerSlice)

		// Create a slice of that type
		sliceType := reflect.SliceOf(innerType)
		return reflect.MakeSlice(sliceType, 0, 0).Interface(), nil

	case ethabi.KindUInt, ethabi.KindInt:
		return []*big.Int{}, nil
	case ethabi.KindAddress:
		return []common.Address{}, nil
	case ethabi.KindBool:
		return []bool{}, nil
	case ethabi.KindString:
		return []string{}, nil
	case ethabi.KindBytes:
		return [][]byte{}, nil
	default:
		return nil, fmt.Errorf("unsupported array element type: %s", elemType.String())
	}
}

// buildTypedSlice builds a properly typed slice from interface elements
func buildTypedSlice(elements []interface{}, elemType *ethabi.Type) (interface{}, error) {
	switch elemType.Kind() {
	case ethabi.KindSlice, ethabi.KindArray:
		// Use reflection to create a slice of the correct type
		// Get an empty slice of the element type to determine the correct Go type
		emptySlice, err := makeEmptySlice(elemType)
		if err != nil {
			return nil, err
		}

		// emptySlice is []Element. e.g. for uint256[][], it is [][]*big.Int (empty)
		sliceType := reflect.TypeOf(emptySlice)

		slice := reflect.MakeSlice(sliceType, len(elements), len(elements))
		for i, e := range elements {
			slice.Index(i).Set(reflect.ValueOf(e))
		}
		return slice.Interface(), nil

	case ethabi.KindUInt, ethabi.KindInt:
		result := make([]*big.Int, len(elements))
		for i, e := range elements {
			switch v := e.(type) {
			case *big.Int:
				result[i] = v
			case uint8:
				result[i] = big.NewInt(int64(v))
			case uint16:
				result[i] = big.NewInt(int64(v))
			case uint32:
				result[i] = big.NewInt(int64(v))
			case uint64:
				result[i] = new(big.Int).SetUint64(v)
			default:
				return nil, fmt.Errorf("unexpected type for uint element: %T", e)
			}
		}
		return result, nil

	case ethabi.KindAddress:
		result := make([]common.Address, len(elements))
		for i, e := range elements {
			result[i] = e.(common.Address)
		}
		return result, nil

	case ethabi.KindBool:
		result := make([]bool, len(elements))
		for i, e := range elements {
			result[i] = e.(bool)
		}
		return result, nil

	case ethabi.KindString:
		result := make([]string, len(elements))
		for i, e := range elements {
			result[i] = e.(string)
		}
		return result, nil

	case ethabi.KindBytes:
		result := make([][]byte, len(elements))
		for i, e := range elements {
			result[i] = e.([]byte)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported array element type: %s", elemType.String())
	}
}

// parseTupleArg parses a tuple argument like "(100,0xabc...,true)"
// Tuple fields are separated by commas and enclosed in parentheses.
func parseTupleArg(val string, typ *ethabi.Type) (interface{}, error) {
	val = strings.TrimSpace(val)
	if !strings.HasPrefix(val, "(") || !strings.HasSuffix(val, ")") {
		return nil, fmt.Errorf("tuple must be enclosed in parentheses: %s", val)
	}

	inner := val[1 : len(val)-1]
	elements := typ.TupleElems()
	if inner == "" && len(elements) == 0 {
		// Empty tuple
		return struct{}{}, nil
	}

	// Split by comma, respecting nested parentheses and brackets
	parts := splitTupleArgs(inner)

	if len(parts) != len(elements) {
		return nil, fmt.Errorf("tuple field count mismatch: expected %d, got %d", len(elements), len(parts))
	}

	// Parse each field
	fields := make([]interface{}, len(parts))
	for i, part := range parts {
		fieldType := elements[i]
		field, err := parseArg(strings.TrimSpace(part), fieldType.Elem)
		if err != nil {
			fieldName := ""
			fieldName = fieldType.Name
			return nil, fmt.Errorf("tuple field %d (%s): %w", i, fieldName, err)
		}
		fields[i] = field
	}

	return fields, nil
}

// splitTupleArgs splits tuple arguments by comma while respecting nested parentheses and brackets
func splitTupleArgs(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, c := range s {
		switch c {
		case '(', '[':
			depth++
			current.WriteRune(c)
		case ')', ']':
			depth--
			current.WriteRune(c)
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(c)
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

// Unused but might be useful for tuple support in future
var _ = strconv.Atoi
