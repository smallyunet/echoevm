package vm

import "testing"

func TestInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	opInvalid(nil, 0xff)
}
