// Package bulkhead provides Netflix/Hystrix-style concurrency isolation and backpressure for Go.
// It partitions execution limits and queue depths to protect critical services from cascading failures.
package bulkhead

import "context"

// Version of the go-bulkhead package.
const Version = "1.0.0"

// ExecuteWithResult executes a function returning a typed value within the bulkhead partition.
func ExecuteWithResult[T any](b *Bulkhead, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	err := b.Execute(ctx, func() error {
		val, fnErr := fn()
		result = val
		return fnErr
	})
	return result, err
}
