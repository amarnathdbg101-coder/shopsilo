# go-studiocomp

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-studiocomp.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-studiocomp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

High-performance e-commerce product photo studio compositor engine in pure Go.

Transforms raw AI cutout masks or product photos into studio-grade catalog assets with automatic edge feathering, grounding shadows, and theme backdrops. Zero cgo. Zero external dependencies.

---

## Features
- **Pure Go Standard Library**: No ImageMagick, no libvips, no C compilers needed.
- **Auto Opaque Cropping**: Detects exact bounding boxes of products and trims transparent margins.
- **Studio Grounding Shadows**: Renders realistic 2.5D elliptical contact and diffused ambient floor shadows.
- **Sub-pixel Edge Feathering**: 3x3 alpha neighborhood smoothing eliminates jagged cutout borders.
- **Category-Tailored Themes**:
  - `ThemeCosmetics`: Luxury soft pearl canvas (`#F6F8FB`) + crisp ground reflection.
  - `ThemeEthnicFashion`: Warm aesthetic gallery canvas (`#FAF9F6`) + soft diffused ambient shadow.
  - `ThemePackagedRetail`: Crisp pure catalog white (`#FFFFFF`) + tight base contact shadow.
  - `ThemeMinimalistDark`: Obsidian charcoal dark mode (`#18181B`).
- **Sub-30ms Execution**: Composites full square catalog imagery in milliseconds.

---

## Installation
```bash
go get github.com/amarnathdbg101-coder/go/go-studiocomp@v1.0.0
```

---

## Quick Start
```go
package main

import (
    "os"
    "github.com/amarnathdbg101-coder/go/go-studiocomp"
)

func main() {
    cutoutBytes, _ := os.ReadFile("product_cutout.png")

    rendered, err := studiocomp.RenderProductStudio(cutoutBytes, studiocomp.Options{
        Theme:       studiocomp.ThemePackagedRetail,
        CanvasSize:  1080,
        PaddingRatio: 0.10,
        JpegQuality: 90,
    })
    if err != nil {
        panic(err)
    }

    _ = os.WriteFile("catalog_product.jpg", rendered, 0644)
}
```

---

## License
MIT
