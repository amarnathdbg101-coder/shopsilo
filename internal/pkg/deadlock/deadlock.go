// Package deadlock provides drop-in replacements for sync.Mutex and sync.RWMutex
// with automatic runtime detection of potential AB-BA lock inversion deadlocks, recursive locks,
// and lock starvation with call stack dumps.
package deadlock

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// Version of the go-deadlock package.
const Version = "1.0.0"

// Mutex is a drop-in replacement for sync.Mutex that tracks lock-acquisition order to detect deadlocks.
type Mutex struct {
	mu sync.Mutex
}

// Lock locks m. If the lock is already in use or if acquiring it would cause a potential deadlock
// cycle, it diagnoses the conflict and reports full caller stack traces.
func (m *Mutex) Lock() {
	cfg := GetConfig()
	if cfg.Disabled {
		m.mu.Lock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(m))
	stack := captureStack(2, cfg.MaxStackDepth)

	globalGraph.preLock(gid, ptr, stack, false)

	var timer *time.Timer
	if cfg.Timeout > 0 {
		timer = time.AfterFunc(cfg.Timeout, func() {
			msg := fmt.Sprintf(
				"Lock acquisition timed out (> %v) on Goroutine %d for Mutex 0x%x!\n"+
					"Waiting caller stack:\n%s",
				cfg.Timeout, gid, ptr, stack,
			)
			if cfg.OnTimeout != nil {
				cfg.OnTimeout(msg)
			}
		})
		defer timer.Stop()
	}

	m.mu.Lock()
	globalGraph.postLock(gid, ptr)
}

// Unlock unlocks m.
func (m *Mutex) Unlock() {
	cfg := GetConfig()
	if cfg.Disabled {
		m.mu.Unlock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(m))
	globalGraph.postUnlock(gid, ptr)
	m.mu.Unlock()
}

// TryLock attempts to lock m without blocking. Returns true if lock was acquired.
func (m *Mutex) TryLock() bool {
	cfg := GetConfig()
	if cfg.Disabled {
		return m.mu.TryLock()
	}

	if !m.mu.TryLock() {
		return false
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(m))
	globalGraph.postLock(gid, ptr)
	return true
}

// RWMutex is a drop-in replacement for sync.RWMutex with deadlock cycle tracking.
type RWMutex struct {
	rw sync.RWMutex
}

// Lock locks rw for writing.
func (rw *RWMutex) Lock() {
	cfg := GetConfig()
	if cfg.Disabled {
		rw.rw.Lock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(rw))
	stack := captureStack(2, cfg.MaxStackDepth)

	globalGraph.preLock(gid, ptr, stack, false)

	var timer *time.Timer
	if cfg.Timeout > 0 {
		timer = time.AfterFunc(cfg.Timeout, func() {
			msg := fmt.Sprintf(
				"RWMutex Write-Lock acquisition timed out (> %v) on Goroutine %d for Mutex 0x%x!\nStack:\n%s",
				cfg.Timeout, gid, ptr, stack,
			)
			if cfg.OnTimeout != nil {
				cfg.OnTimeout(msg)
			}
		})
		defer timer.Stop()
	}

	rw.rw.Lock()
	globalGraph.postLock(gid, ptr)
}

// Unlock unlocks rw for writing.
func (rw *RWMutex) Unlock() {
	cfg := GetConfig()
	if cfg.Disabled {
		rw.rw.Unlock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(rw))
	globalGraph.postUnlock(gid, ptr)
	rw.rw.Unlock()
}

// RLock locks rw for reading.
func (rw *RWMutex) RLock() {
	cfg := GetConfig()
	if cfg.Disabled {
		rw.rw.RLock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(rw))
	stack := captureStack(2, cfg.MaxStackDepth)

	globalGraph.preLock(gid, ptr, stack, true)
	rw.rw.RLock()
	globalGraph.postLock(gid, ptr)
}

// RUnlock undoes a single RLock call.
func (rw *RWMutex) RUnlock() {
	cfg := GetConfig()
	if cfg.Disabled {
		rw.rw.RUnlock()
		return
	}

	gid := curGID()
	ptr := uintptr(unsafe.Pointer(rw))
	globalGraph.postUnlock(gid, ptr)
	rw.rw.RUnlock()
}
