# SPEC: go-imghash

## 1. Scope & Purpose
`go-imghash` generates 64-bit gradient Difference Hashes (dHash) from image streams to detect duplicate, resized, compressed, or slightly modified images. It replaces heavy C-based image processing tools with a pure Go solution.

## 2. Public API Contract
- `type Hasher interface`: Contract for computing perceptual hashes.
- `func Compute(r io.Reader) (string, error)`: Stream reader returning 16-character hex dHash.
- `func ComputeImage(img image.Image) (string, error)`: Generates dHash from in-memory `image.Image`.
- `func Distance(hex1, hex2 string) (int, error)`: Computes bitwise Hamming distance (0 to 64).
- `func Similar(hex1, hex2 string, maxDistance ...int) bool`: Threshold-based similarity boolean.

## 3. Concurrency & Memory Guarantees
- Zero reflection: Uses direct array indexing for downscaled 9x8 matrix.
- Allocation efficiency: Fixed matrix `[8][9]uint8` allocated on stack.
- Safe for concurrent use across multiple goroutines.

## 4. How to Extend
Additional perceptual algorithms (such as pHash DCT or aHash Average Hash) can implement the `Hasher` interface.
