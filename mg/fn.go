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

// F takes a function that is compatible as a mage target, and any args that need to be passed to
// it, and wraps it in an mg.Function that mg.Deps can run. Args must be passed in the same order as they
// are declared by the function. Note that you do not need to and should not pass a context.Context
// to F, even if the target takes a context. Compatible args are int, bool, string, and
// time.Duration.
func F(target interface{}, args ...interface{}) Function {
	hasContext, isNamespace, err := checkF(target, args)
	if err != nil {
		panic(err)
	}
	return fn{
		name: runtime.FuncForPC(reflect.ValueOf(target).Pointer()).Name(),
		f: func(ctx context.Context) error {
			v := reflect.ValueOf(target)
			count := len(args)
			if hasContext {
				count++
			}
			if isNamespace {
				count++
			}
			vargs := make([]reflect.Value, count)
			x := 0
			if isNamespace {
				vargs[0] = reflect.ValueOf(struct{}{})
				x++
			}
			if hasContext {
				vargs[x] = reflect.ValueOf(ctx)
				x++
			}
			for y := range args {
				vargs[x+y] = reflect.ValueOf(args[y])
			}
			ret := v.Call(vargs)
			if len(ret) > 0 {
				// we only allow functions with a single error return, so this should be safe.
				if ret[0].IsNil() {
					return nil
				}
				return ret[0].Interface().(error)
			}
			return nil
		},
	}
}

type fn struct {
	name string
	f    func(ctx context.Context) error
}

// Name returns the fully qualified name of the function.
func (f fn) Name() string {
	return f.name
}

// Run runs the function.
func (f fn) Run(ctx context.Context) error {
	return f.f(ctx)
}

func checkF(target interface{}, args []interface{}) (hasContext, isNamespace bool, _ error) {
	t := reflect.TypeOf(target)
	if t == nil || t.Kind() != reflect.Func {
		return false, false, fmt.Errorf("non-function passed to mg.F: %T", target)
	}
	if t.NumOut() > 1 {
		return false, false, fmt.Errorf("target has too many return values, must be zero or just an error: %T", target)
	}
	if t.NumOut() == 1 && t.Out(0) != reflect.TypeOf(func() error { return nil }).Out(0) {
		return false, false, fmt.Errorf("target's return value is not an error")
	}
	// more inputs than slots is always an error
	if len(args) > t.NumIn() {
		return false, false, fmt.Errorf("too many arguments for target, got %d for %T", len(args), target)
	}
	if t.NumIn() == 0 {
		return false, false, nil
	}
	x := 0
	inputs := t.NumIn()
	if t.In(0).AssignableTo(reflect.TypeOf(struct{}{})) {
		// nameSpace func
		isNamespace = true
		x++
		// callers must leave off the namespace value
		inputs--
	}
	if t.NumIn() > x && t.In(x) == reflect.TypeOf(func(context.Context) {}).In(0) {
		// callers must leave off the context
		inputs--
		// let the upper function know it should pass us a context.
		hasContext = true
		// skip checking the first argument in the below loop if it's a context, since first arg is
		// special.
		x++
	}
	if len(args) != inputs {
		return false, false, fmt.Errorf("wrong number of arguments for target, got %d for %T", len(args), target)
	}
	for _, arg := range args {
		argT := t.In(x)
		switch argT {
		case reflect.TypeOf(0), reflect.TypeOf(""), reflect.TypeOf(false):
			// ok
		default:
			return false, false, fmt.Errorf("argument %d (%s), is not a supported argument type", x, argT)
		}
		passedT := reflect.TypeOf(arg)
		if argT != passedT {
			return false, false, fmt.Errorf("argument %d expected to be %s, but is %s", x, argT, passedT)
		}
		x++
	}
	return hasContext, isNamespace, nil
}
