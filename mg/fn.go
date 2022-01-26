package mg

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
)

// Function represents a function that can be run with mg.Deps.
type Function interface {
	// Name is a unique identifier and display name for the function.
	Name() string

	// Run the function.
	Run(ctx context.Context) error
}

// Fn creates a Function from a compatible target function and args.
func Fn(target interface{}, args ...interface{}) Function {
	result, err := newFn(target, args...)
	if err != nil {
		panic(err)
	}
	return result
}

func newFn(f interface{}, args ...interface{}) (Function, error) {
	v := reflect.ValueOf(f)
	if f == nil || v.Type().Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function passed to mg.Fn: %T", f)
	}
	if v.Type().NumOut() != 1 || v.Type().Out(0) != reflect.TypeOf(func() error { return nil }).Out(0) {
		return nil, fmt.Errorf("function does not have an error return value: %T", f)
	}
	if len(args) > v.Type().NumIn() {
		return nil, fmt.Errorf("too many arguments %d for function %T", len(args), f)
	}
	var hasNamespace bool
	x := 0
	inputs := v.Type().NumIn()
	if v.Type().In(0).AssignableTo(reflect.TypeOf(struct{}{})) {
		hasNamespace = true
		x++
		inputs--
	}
	if v.Type().NumIn() > x && v.Type().In(x) == reflect.TypeOf(func(context.Context) {}).In(0) {
		inputs--
		x++
	} else {
		return nil, fmt.Errorf("invalid function, must have context.Context as first argument")
	}
	if len(args) != inputs {
		return nil, fmt.Errorf("wrong number of arguments for fn, got %d for %T", len(args), f)
	}
	for _, arg := range args {
		argT := v.Type().In(x)
		switch argT {
		case reflect.TypeOf(0), reflect.TypeOf(""), reflect.TypeOf(false):
			// ok
		default:
			return nil, fmt.Errorf("argument %d (%s), is not a supported argument type", x, argT)
		}
		if callArgT := reflect.TypeOf(arg); argT != callArgT {
			return nil, fmt.Errorf("argument %d expected to be %s, but is %s", x, argT, callArgT)
		}
		x++
	}
	argCount := len(args) + 1 // +1 for context
	if hasNamespace {
		argCount++ // +1 for the namespace
	}
	return fn{
		name: runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(),
		f: func(ctx context.Context) error {
			callArgs := make([]reflect.Value, 0, argCount)
			if hasNamespace {
				callArgs = append(callArgs, reflect.ValueOf(struct{}{}))
			}
			callArgs = append(callArgs, reflect.ValueOf(ctx))
			for _, arg := range args {
				callArgs = append(callArgs, reflect.ValueOf(arg))
			}
			ret := v.Call(callArgs)
			if ret[0].IsNil() {
				return nil
			}
			return ret[0].Interface().(error)
		},
	}, nil
}

type fn struct {
	name string
	f    func(ctx context.Context) error
}

// Name implements Function.
func (f fn) Name() string {
	return f.name
}

// Run implements Function.
func (f fn) Run(ctx context.Context) error {
	return f.f(ctx)
}
