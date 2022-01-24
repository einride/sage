package runner

import (
	"context"
	"sync"
)

// global state for the runner.
// nolint: gochecknoglobals
var (
	mu      sync.Mutex
	onceFns = map[string]func(context.Context) error{}
)

// RunOnce uses key to ensure that fn runs exactly once and always returns the error from the initial run.
func RunOnce(ctx context.Context, key string, fn func(context.Context) error) error {
	mu.Lock()
	onceFn, ok := onceFns[key]
	if !ok {
		onceFn = makeOnceFn(fn)
		onceFns[key] = onceFn
	}
	mu.Unlock()
	return onceFn(ctx)
}

func makeOnceFn(fn func(context.Context) error) func(context.Context) error {
	var once sync.Once
	var err error
	return func(ctx context.Context) error {
		once.Do(func() {
			err = fn(ctx)
		})
		return err
	}
}
