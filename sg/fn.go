package sg

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
)

// Target represents a target function that can be run with Deps.
type Target interface {
	// Name is a non-unique display name for the Target.
	Name() string

	// ID is a unique identifier for the Target.
	ID() string

	// Run the Target.
	Run(ctx context.Context) error
}

// Fn creates a Target from a compatible function and args.
func Fn(target interface{}, args ...interface{}) Target {
	result, err := newFn(target, args...)
	if err != nil {
		panic(err)
	}
	return result
}

func newFn(f interface{}, args ...interface{}) (Target, error) {
	v := reflect.ValueOf(f)
	if f == nil || v.Type().Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function passed to sg.Fn: %T", f)
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
	argsID, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON name for args: %w", err)
	}
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	return fn{
		name: name,
		id:   name + "(" + string(argsID) + ")",
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
	id   string
	f    func(ctx context.Context) error
}

// ID implements Target.
func (f fn) ID() string {
	return f.id
}

// Name implements Target.
func (f fn) Name() string {
	return f.name
}

// Run implements Target.
func (f fn) Run(ctx context.Context) error {
	return f.f(ctx)
}
