# go-imagefx

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-imagefx.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-imagefx)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Pure Go e-commerce product studio compositor, adaptive boundary background cutouts, soft drop-shadow generation, and canvas aspect-ratio fitting with zero CGO or third-party dependencies.

---

## Features
- **Adaptive Boundary Flood Cutout**: Perimeter sampling & BFS flood fill with soft edge feathering.
- **Bounding Box Auto-Trim**: Automatically crops away empty transparent margins.
- **Aspect-Ratio Canvas Fit**: Centers product cutouts onto square 1:1 e-commerce canvases (e.g. 800x800 white background).
- **Soft Drop Shadows**: Synthetic separable box-blur drop shadow renderer.
- **Bilinear Resizer**: High-quality pure Go thumbnail downscaler.
- **Zero Third-Party Dependencies**: 100% standard library (`image`, `image/color`, `image/draw`, `math`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-imagefx@v1.0.0
```

---

## Quickstart

```go
// 1. Cut out background
cutout, ok := goimagefx.AdaptiveBoundaryFloodCutout(srcImg, 42.0)

// 2. Crop excess transparent borders
trimmed, _ := goimagefx.TrimTransparency(cutout, 128)

// 3. Add soft drop shadow
withShadow := goimagefx.ApplyDropShadow(trimmed, 8, 4, 8, 110)

// 4. Center on 800x800 square white canvas with 10% padding
studioShot := goimagefx.FitAndCenter(withShadow, 800, 800, 0.10, color.White)
```

---

## License
MIT License.
