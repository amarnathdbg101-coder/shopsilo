# SPEC: go-bizhours

## 1. Scope & Purpose
`go-bizhours` calculates real-time operating availability, countdowns, and customer banner messages for stores, restaurants, hospitals, and support desks. It seamlessly handles daytime shifts, overnight shifts spanning midnight, 24-hour operations, weekly holidays, and emergency manual toggles.

## 2. Public API Contract
- `type LiveStatus struct`: Status payload containing flags, remaining minutes, and formatted banner text.
- `func Calculate(isOpenManual bool, openTime, closeTime, weeklyOff string) LiveStatus`: Evaluates against current system time.
- `func CalculateAt(isOpenManual bool, openTime, closeTime, weeklyOff string, target time.Time) LiveStatus`: Deterministic calculation against reference time.
- `func FormatTime(timeStr string) string`: Formats 24-hr time strings into user-friendly 12-hr AM/PM strings.

## 3. Concurrency & Memory Guarantees
- Zero reflection and zero memory allocations on hot paths.
- Deterministic stateless calculations.
- Safe for concurrent read access across unbounded goroutines.

## 4. How to Extend
Multi-shift support (e.g. morning + evening shifts with afternoon break) can be configured by evaluating intervals sequentially.
