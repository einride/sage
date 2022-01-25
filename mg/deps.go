package mg

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/go-logr/logr"
)

// Deps runs the given functions as dependencies of the calling function in parallel.
//
// Dependencies must only be of type func(context.Context) error.
// Or a similar method on a mg.Namespace type.
// Or an mg.Fn interface.
//
// The function calling Deps is guaranteed that all dependent functions will be
// run exactly once when Deps returns.  Dependent functions may in turn declare
// their own dependencies using Deps. Each dependency is run in their own
// goroutines. Each function is given the context provided if the function
// prototype allows for it.
func Deps(ctx context.Context, fns ...interface{}) {
	var mu sync.Mutex
	errs := make(map[string]error)
	var wg sync.WaitGroup
	for _, f := range checkFns(fns) {
		fn := onces.LoadOrStore(f)
		wg.Add(1)
		go func() {
			ctx = logr.NewContext(ctx, NewLogger(fn.fn.Name()))
			defer func() {
				if v := recover(); v != nil {
					errs[fn.fn.Name()] = fmt.Errorf(fmt.Sprint(v))
					mu.Unlock()
				}
				wg.Done()
			}()
			if err := fn.run(ctx); err != nil {
				mu.Lock()
				errs[fn.fn.Name()] = err
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(errs) > 0 {
		for name, err := range errs {
			NewLogger(name).Error(err, err.Error())
		}
		os.Exit(1)
	}
}

// SerialDeps works like Deps except running all dependencies serially instead of in parallel.
func SerialDeps(ctx context.Context, fns ...interface{}) {
	for _, fn := range fns {
		Deps(ctx, fn)
	}
}

type onceMap struct {
	mu sync.Mutex
	m  map[string]*onceFun
}

func (o *onceMap) LoadOrStore(f Fn) *onceFun {
	defer o.mu.Unlock()
	o.mu.Lock()
	existing, ok := o.m[f.Name()]
	if ok {
		return existing
	}
	one := &onceFun{
		fn: f,
	}
	o.m[f.Name()] = one
	return one
}

// nolint: gochecknoglobals
var onces = &onceMap{
	m: map[string]*onceFun{},
}

func checkFns(fns []interface{}) []Fn {
	funcs := make([]Fn, len(fns))
	for i, f := range fns {
		if fn, ok := f.(Fn); ok {
			funcs[i] = fn
			continue
		}
		// Check if the target provided is a not function so we can give a clear warning
		t := reflect.TypeOf(f)
		if t == nil || t.Kind() != reflect.Func {
			panic(fmt.Errorf("non-function used as a target dependency: %T", f))
		}
		funcs[i] = F(f)
	}
	return funcs
}

type onceFun struct {
	once sync.Once
	fn   Fn
	err  error
}

// run will run the function exactly once and capture the error output. Further runs simply return
// the same error output.
func (o *onceFun) run(ctx context.Context) error {
	o.once.Do(func() {
		o.err = o.fn.Run(ctx)
	})
	return o.err
}
