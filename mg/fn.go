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

func newFn(target interface{}, args ...interface{}) (Function, error) {
	t := reflect.TypeOf(target)
	if t == nil || t.Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function passed to mg.Fn: %T", target)
	}
	if t.NumOut() > 1 {
		return nil, fmt.Errorf("target has too many return values, must be zero or just an error: %T", target)
	}
	if t.NumOut() == 1 && t.Out(0) != reflect.TypeOf(func() error { return nil }).Out(0) {
		return nil, fmt.Errorf("target's return value is not an error")
	}
	// more inputs than slots is always an error
	if len(args) > t.NumIn() {
		return nil, fmt.Errorf("too many arguments for target, got %d for %T", len(args), target)
	}
	var hasNamespace bool
	x := 0
	inputs := t.NumIn()
	if t.In(0).AssignableTo(reflect.TypeOf(struct{}{})) {
		hasNamespace = true
		x++
		// callers must leave off the namespace value
		inputs--
	}
	if t.NumIn() > x && t.In(x) == reflect.TypeOf(func(context.Context) {}).In(0) {
		// callers must leave off the context
		inputs--
		// skip checking the first argument in the below loop if it's a context, since first arg is
		// special.
		x++
	} else {
		return nil, fmt.Errorf("invalid function, must have context.Context as first argument")
	}
	if len(args) != inputs {
		return nil, fmt.Errorf("wrong number of arguments for target, got %d for %T", len(args), target)
	}
	for _, arg := range args {
		argT := t.In(x)
		switch argT {
		case reflect.TypeOf(0), reflect.TypeOf(""), reflect.TypeOf(false):
			// ok
		default:
			return nil, fmt.Errorf("argument %d (%s), is not a supported argument type", x, argT)
		}
		passedT := reflect.TypeOf(arg)
		if argT != passedT {
			return nil, fmt.Errorf("argument %d expected to be %s, but is %s", x, argT, passedT)
		}
		x++
	}
	return fn{
		name: runtime.FuncForPC(reflect.ValueOf(target).Pointer()).Name(),
		f: func(ctx context.Context) error {
			v := reflect.ValueOf(target)
			count := len(args) + 1
			if hasNamespace {
				count++
			}
			vargs := make([]reflect.Value, count)
			x := 0
			if hasNamespace {
				vargs[0] = reflect.ValueOf(struct{}{})
				x++
			}
			vargs[x] = reflect.ValueOf(ctx)
			x++
			for y := range args {
				vargs[x+y] = reflect.ValueOf(args[y])
			}
			ret := v.Call(vargs)
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
