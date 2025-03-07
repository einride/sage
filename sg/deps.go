package sg

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"go.einride.tech/sage/sg/internal/runner"
)

// Deps runs each of the provided functions in parallel.
//
// Dependencies must be of type func(context.Context) error or Target.
//
// Each function will be run exactly once, even across multiple calls to Deps.
func Deps(ctx context.Context, functions ...interface{}) {
	errs := make([]error, len(functions))
	checkedFunctions := checkFunctions(functions...)
	var wg sync.WaitGroup
	for i, f := range checkedFunctions {
		dependencies := getDependencies(ctx)
		for _, dependency := range dependencies {
			if dependency.ID() == f.ID() {
				depNames := make([]string, len(dependencies))
				for i, dependency := range dependencies {
					depNames[i] = dependency.Name()
				}
				msg := fmt.Sprintf("dependency cycle calling %s! chain: %s", f.Name(), strings.Join(depNames, ","))
				panic(msg)
			}
		}
		ctx := withDependency(ctx, f)

		// Forcing serial deps can protect low-powered build machines from running out of memory.
		// EXPERIMENTAL: Support for this environment variable may be removed at any time.
		if forceSerialDeps, ok := os.LookupEnv("SAGE_FORCE_SERIAL_DEPS"); ok && isTrue(forceSerialDeps) {
			errs[i] = runner.RunOnce(WithLogger(ctx, NewLogger(f.Name())), f.ID(), f.Run)
			continue
		}
		wg.Add(1)
		go func() {
			defer func() {
				if v := recover(); v != nil {
					errs[i] = fmt.Errorf("%s", v)
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

func checkFunctions(functions ...interface{}) []Target {
	result := make([]Target, 0, len(functions))
	for _, f := range functions {
		if checked, ok := f.(Target); ok {
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

func isTrue(s string) bool {
	value, err := strconv.ParseBool(s)
	return err == nil && value
}

type dependencyChainContextKey struct{}

func getDependencies(ctx context.Context) []Target {
	result, ok := ctx.Value(dependencyChainContextKey{}).([]Target)
	if !ok {
		return []Target{}
	}
	return result
}

func withDependency(ctx context.Context, target Target) context.Context {
	dependencies := getDependencies(ctx)
	dependencies = append(dependencies, target)
	return context.WithValue(ctx, dependencyChainContextKey{}, dependencies)
}
