package sg

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"go.einride.tech/sage/sg/internal/runner"
)

// Deps runs each of the provided functions in parallel.
//
// Dependencies must be of type func(context.Context) error or Function.
//
// Each function will be run exactly once, even across multiple calls to Deps.
func Deps(ctx context.Context, functions ...interface{}) {
	errs := make([]error, len(functions))
	checkedFunctions := checkFunctions(functions...)
	var wg sync.WaitGroup
	for i, f := range checkedFunctions {
		i, f := i, f
		wg.Add(1)
		go func() {
			defer func() {
				if v := recover(); v != nil {
					errs[i] = fmt.Errorf(fmt.Sprint(v))
				}
				wg.Done()
			}()
			errs[i] = runner.RunOnce(WithLogger(ctx, NewLogger(f.Name())), f.ID(), f.Run)
		}()
	}
	wg.Wait()
	var exitError bool
	for i, err := range errs {
		if err != nil {
			NewLogger(checkedFunctions[i].Name()).Println(err)
			exitError = true
		}
	}
	if exitError {
		os.Exit(1)
	}
}

// SerialDeps works like Deps except running all dependencies serially instead of in parallel.
func SerialDeps(ctx context.Context, targets ...interface{}) {
	for _, target := range targets {
		Deps(ctx, target)
	}
}

func checkFunctions(functions ...interface{}) []Function {
	result := make([]Function, 0, len(functions))
	for _, f := range functions {
		if checked, ok := f.(Function); ok {
			result = append(result, checked)
			continue
		}
		t := reflect.TypeOf(f)
		if t == nil || t.Kind() != reflect.Func {
			panic(fmt.Errorf("non-function used as a target dependency: %T", f))
		}
		result = append(result, Fn(f))
	}
	return result
}
