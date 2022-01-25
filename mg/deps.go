package mg

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mglogr"
)

type onceMap struct {
	mu *sync.Mutex
	m  map[onceKey]*onceFun
}

type onceKey struct {
	Name string
	ID   string
}

func (o *onceMap) LoadOrStore(f Fn) *onceFun {
	defer o.mu.Unlock()
	o.mu.Lock()

	key := onceKey{
		Name: f.Name(),
		ID:   f.ID(),
	}
	existing, ok := o.m[key]
	if ok {
		return existing
	}
	one := &onceFun{
		once:        &sync.Once{},
		fn:          f,
		displayName: displayName(f.Name()),
	}
	o.m[key] = one
	return one
}

// nolint: gochecknoglobals
var onces = &onceMap{
	mu: &sync.Mutex{},
	m:  map[onceKey]*onceFun{},
}

// SerialDeps is like Deps except it runs each dependency serially, instead of
// in parallel. This can be useful for resource intensive dependencies that
// shouldn't be run at the same time.
func SerialDeps(ctx context.Context, fns ...interface{}) {
	funcs := checkFns(fns)
	for i := range fns {
		runDeps(ctx, funcs[i:i+1])
	}
}

// Deps runs the given functions as dependencies of the calling function.
// Dependencies must only be of type:
//     func()
//     func() error
//     func(context.Context)
//     func(context.Context) error
// Or a similar method on a mg.Namespace type.
// Or an mg.Fn interface.
//
// The function calling Deps is guaranteed that all dependent functions will be
// run exactly once when Deps returns.  Dependent functions may in turn declare
// their own dependencies using Deps. Each dependency is run in their own
// goroutines. Each function is given the context provided if the function
// prototype allows for it.
func Deps(ctx context.Context, fns ...interface{}) {
	funcs := checkFns(fns)
	runDeps(ctx, funcs)
}

// runDeps assumes you've already called checkFns.
func runDeps(ctx context.Context, fns []Fn) {
	mu := &sync.Mutex{}
	errs := make(map[string]error)
	wg := &sync.WaitGroup{}
	for _, f := range fns {
		fn := onces.LoadOrStore(f)
		wg.Add(1)
		go func() {
			ctx = logr.NewContext(ctx, mglogr.New(fn.displayName))
			defer func() {
				if v := recover(); v != nil {
					errs[fn.displayName] = fmt.Errorf(fmt.Sprint(v))
					mu.Unlock()
				}
				wg.Done()
			}()
			if err := fn.run(ctx); err != nil {
				mu.Lock()
				errs[fn.displayName] = err
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		for name, err := range errs {
			mglogr.New(name).Error(err, err.Error())
		}
		os.Exit(1)
	}
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

// funcName returns the unique name for the function.
func funcName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func displayName(name string) string {
	splitByPackage := strings.Split(name, ".")
	if len(splitByPackage) == 2 && splitByPackage[0] == "main" {
		return splitByPackage[len(splitByPackage)-1]
	}
	return name
}

type onceFun struct {
	once *sync.Once
	fn   Fn
	err  error

	displayName string
}

// run will run the function exactly once and capture the error output. Further runs simply return
// the same error output.
func (o *onceFun) run(ctx context.Context) error {
	o.once.Do(func() {
		o.err = o.fn.Run(ctx)
	})
	return o.err
}
