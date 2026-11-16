# Specification: LMAX Disruptor Ring Buffer (`go-disruptor`)

## 1. Hardware Architecture Invariants
- **Cache-Line Padding**:
  A modern x86/ARM64 CPU core loads memory in 64-byte chunks.
  $$\text{sizeof}(\text{Sequence}) = 7 \times 8\text{B} + 8\text{B} + 7 \times 8\text{B} = 120\text{ bytes} \ge 64\text{ bytes}$$
  Guarantees producer and consumer sequence counters reside on strictly isolated CPU cache lines.
- **Power of Two Invariant**:
  $$\text{Index} = \text{Sequence} \ \& \ (\text{Capacity} - 1)$$
  Capacity $C$ must satisfy $C = 2^k, k \in [1, 30]$.

## 2. Concurrency Guarantees
- Single Producer / Multi-Consumer sequence barriers prevent overwriting unconsumed slots:
  $$\text{Cursor} - \text{GatingSequence} \le \text{Capacity}$$
- Monotonic sequence numbers guarantee total order of event dispatch.
