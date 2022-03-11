package sg_test

import (
	"context"
	"testing"

	"go.einride.tech/sage/sg"
)

func TestFn(t *testing.T) {
	ctx := context.Background()

	sg.Deps(ctx, sg.Fn(FuncWithInt, 1))
	shouldPanic(t, func() { sg.Deps(ctx, sg.Fn(FuncWithInt, "string")) })

	sg.Deps(ctx, sg.Fn(FuncWithString, "one"))
	shouldPanic(t, func() { sg.Deps(ctx, sg.Fn(FuncWithString, 1)) })

	sg.Deps(ctx, sg.Fn(FuncWithBool, true))
	shouldPanic(t, func() { sg.Deps(ctx, sg.Fn(FuncWithBool, "string")) })

	s := S{One: "one"}
	sg.Deps(ctx, sg.Fn(FuncWithStruct, s))
	shouldPanic(t, func() { sg.Deps(ctx, sg.Fn(FuncWithStruct, "string")) })
}

func Func(ctx context.Context) error {
	return nil
}

func FuncWithInt(ctx context.Context, arg int) error {
	return nil
}

func FuncWithString(ctx context.Context, arg string) error {
	return nil
}

func FuncWithBool(ctx context.Context, arg bool) error {
	return nil
}

type S struct {
	One string
}

func FuncWithStruct(ctx context.Context, arg S) error {
	return nil
}

func shouldPanic(t *testing.T, fn func()) {
	defer func() {
		if p := recover(); p == nil {
			t.Error("there should be a panic")
		}
	}()

	fn()
}
