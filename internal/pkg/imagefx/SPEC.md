# Technical Specification: go-imagefx

## 1. Domain & Purpose
`go-imagefx` is a high-speed, zero-dependency e-commerce product studio image processing engine in pure Go. It provides adaptive boundary flood cutouts (transparent background extraction), transparent border trimming, square canvas centering with aspect-ratio fitting, soft Gaussian-like studio drop shadows, and pure-Go bilinear interpolation image resizing.

## 2. Architecture & Design Principles
- **Adaptive Boundary Flood Cutout**: Samples image perimeter pixels, calculates median RGB values, and executes multi-source breadth-first-search (BFS) color flooding with smooth edge feathering to isolate objects without GPU or cloud latency.
- **Bounding Box Alpha Trimming**: Detects minimum rectangular coordinates enclosing non-transparent pixels (`alpha >= threshold`) to strip empty margins.
- **Square Canvas Fit & Padding**: Standardizes e-commerce catalog images into clean 1:1 squares (e.g., 800x800 or 1000x1000) with configurable padding (default 10%) and customizable background color (white, grey, transparent).
- **Soft Drop Shadow Synthesis**: Computes alpha silhouettes and applies 2-pass separable box blurs to create realistic soft drop shadows under floating products.
- **Pure Go Bilinear Interpolator**: Resizes images without cgo, Libvips, or ImageMagick system dependencies.

## 3. API Surface
- `func AdaptiveBoundaryFloodCutout(src image.Image, tolerance float64) (*image.RGBA, bool)`
- `func TrimTransparency(src image.Image, alphaThreshold uint8) (image.Image, image.Rectangle)`
- `func FitAndCenter(src image.Image, canvasW, canvasH int, paddingPct float64, bg color.Color) *image.RGBA`
- `func ApplyDropShadow(src image.Image, blurRadius int, offsetX, offsetY int, shadowAlpha uint8) *image.RGBA`
- `func ResizeBilinear(src image.Image, targetW, targetH int) *image.RGBA`
