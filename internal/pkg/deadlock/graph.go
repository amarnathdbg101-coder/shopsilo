package deadlock

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
)

// curGID extracts the current goroutine ID from runtime.Stack in pure Go without CGO.
func curGID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	line := buf[:n]
	spaceIdx := bytes.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return 0
	}
	rest := line[spaceIdx+1:]
	endIdx := bytes.IndexByte(rest, ' ')
	if endIdx == -1 {
		return 0
	}
	id, err := strconv.ParseUint(string(rest[:endIdx]), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

type lockGraph struct {
	mu sync.Mutex

	// edges: source lock ptr -> map of target lock ptrs -> stack trace of transition
	edges map[uintptr]map[uintptr]string

	// heldLocks: goroutine ID -> list of lock pointers currently held by that goroutine
	heldLocks map[uint64][]uintptr

	// lockStacks: lock ptr -> stack trace where lock was acquired
	lockStacks map[uintptr]string
}

var globalGraph = &lockGraph{
	edges:      make(map[uintptr]map[uintptr]string),
	heldLocks:  make(map[uint64][]uintptr),
	lockStacks: make(map[uintptr]string),
}

// captureStack captures caller stack frames.
func captureStack(skip int, maxDepth int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var buf bytes.Buffer
	count := 0
	for {
		frame, more := frames.Next()
		if count >= maxDepth {
			break
		}
		if !bytes.Contains([]byte(frame.Function), []byte("go-deadlock")) {
			fmt.Fprintf(&buf, "  %s\n    %s:%d\n", frame.Function, frame.File, frame.Line)
			count++
		}
		if !more {
			break
		}
	}
	return buf.String()
}

// preLock is called before acquiring targetLock. It checks for lock-order inversion cycles.
func (g *lockGraph) preLock(gid uint64, targetLock uintptr, stack string, isRead bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.lockStacks[targetLock] = stack

	held := g.heldLocks[gid]
	for _, sourceLock := range held {
		if sourceLock == targetLock {
			// If it's a re-entrant read lock, that's allowed in RWMutex
			if isRead {
				continue
			}
			// Recursive locking on write mutex is a self-deadlock
			msg := fmt.Sprintf("Recursive locking detected on goroutine %d (mutex 0x%x):\n%s", gid, targetLock, stack)
			if globalConfig.OnPotentialDeadlock != nil {
				globalConfig.OnPotentialDeadlock(msg)
			}
			continue
		}

		// Check if targetLock can reach sourceLock in the lock-order graph
		if cycleStack, hasCycle := g.canReach(targetLock, sourceLock); hasCycle {
			msg := fmt.Sprintf(
				"Potential Lock Inversion (AB-BA Deadlock) detected on Goroutine %d!\n"+
					"Goroutine %d holds lock 0x%x and wants lock 0x%x.\n"+
					"However, another path previously acquired 0x%x before 0x%x:\n"+
					"--- Prior Acquisition Order:\n%s\n"+
					"--- Current Acquisition Attempt:\n%s",
				gid, gid, sourceLock, targetLock, targetLock, sourceLock, cycleStack, stack,
			)
			if globalConfig.OnPotentialDeadlock != nil {
				globalConfig.OnPotentialDeadlock(msg)
			}
		}

		// Record the directed edge: sourceLock -> targetLock
		if _, ok := g.edges[sourceLock]; !ok {
			g.edges[sourceLock] = make(map[uintptr]string)
		}
		g.edges[sourceLock][targetLock] = stack
	}
}

// postLock is called after targetLock has been successfully acquired.
func (g *lockGraph) postLock(gid uint64, targetLock uintptr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.heldLocks[gid] = append(g.heldLocks[gid], targetLock)
}

// postUnlock is called when targetLock is released.
func (g *lockGraph) postUnlock(gid uint64, targetLock uintptr) {
	g.mu.Lock()
	defer g.mu.Unlock()

	held := g.heldLocks[gid]
	for i := len(held) - 1; i >= 0; i-- {
		if held[i] == targetLock {
			g.heldLocks[gid] = append(held[:i], held[i+1:]...)
			break
		}
	}
	if len(g.heldLocks[gid]) == 0 {
		delete(g.heldLocks, gid)
	}
}

// canReach performs a Depth-First Search (DFS) to see if 'to' is reachable from 'from'.
func (g *lockGraph) canReach(from, to uintptr) (string, bool) {
	visited := make(map[uintptr]bool)
	return g.dfs(from, to, visited)
}

func (g *lockGraph) dfs(curr, target uintptr, visited map[uintptr]bool) (string, bool) {
	if curr == target {
		return g.lockStacks[curr], true
	}
	visited[curr] = true

	for neighbor, edgeStack := range g.edges[curr] {
		if !visited[neighbor] {
			if stack, found := g.dfs(neighbor, target, visited); found {
				return edgeStack + "\n" + stack, true
			}
		}
	}
	return "", false
}



