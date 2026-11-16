# go-lru

Thread-safe, generic Least Recently Used (LRU) cache with per-item TTL and eviction callbacks for Go.

---

## Features
- **Strict O(1) Performance**: Instant key lookup and node reordering via internal doubly linked list + hash map.
- **Type-Safe Generics**: Works with any comparable key `K` and value `V` without interface reflection.
- **Per-Item TTL**: Set independent expiration windows per cache entry.
- **Eviction Hook**: Notified via `OnEvict(key, val)` when an element is evicted.
- **Order Inspection**: Inspect entries from Most Recently Used (MRU) to Least Recently Used (LRU) via `Keys()`.
- **Zero External Dependencies**: Pure Go standard library.

---

## Quickstart

```go
package main

import (
	"fmt"
	"time"

	golru "go-lru"
)

func main() {
	// Create cache with capacity 2 and eviction logger
	cache, _ := golru.New(2, func(key string, val string) {
		fmt.Printf("[EVICTED] %s -> %s\n", key, val)
	})

	cache.Set("session:1", "Alice", 0)
	cache.Set("session:2", "Bob", 0)

	// Access session:1 (becomes Most Recently Used)
	val, _ := cache.Get("session:1")
	fmt.Println("Active user:", val)

	// Add 3rd item -> Capacity 2 reached -> session:2 is evicted!
	cache.Set("session:3", "Charlie", 0)

	fmt.Println("Has session:2?", cache.Contains("session:2")) // false
	fmt.Println("Has session:1?", cache.Contains("session:1")) // true
	fmt.Println("Has session:3?", cache.Contains("session:3")) // true
}
```

---

## Testing
Run unit tests with full race and count coverage:
```bash
go test -v -count=1 ./...
```
