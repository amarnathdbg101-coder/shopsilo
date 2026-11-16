# SPEC: `go-studiocomp` — E-commerce Photo Studio Compositor Engine

## 1. Overview & Problem Statement
In e-commerce catalogs (Shopify, Amazon, Flipkart, ONDC, custom store apps), raw camera photos or AI cutout masks (from Cloudflare AI, Rembg, etc.) have:
- Non-standard dimensions and inconsistent product scale.
- Raw transparent PNGs that look "floating" and fake without ground contact shadows.
- Jagged alpha cutout edges from ML segmentation models.
- Wrong background colors inconsistent with catalog branding (e.g. gray, dirty white, checkered transparency).

`go-studiocomp` provides a pure Go, sub-50ms catalog image compositor that:
1. Detects tight opaque bounds around product cutouts (discarding transparent margins).
2. Scales with high-quality bilinear interpolation.
3. Feathers outer alpha edges (1-pixel 3x3 smoothing) to eliminate pixelation and jagged ML artifacts.
4. Generates realistic 2.5D elliptical contact and diffused ambient shadows matching real studio lighting physics.
5. Insets the product with optimal breathing room (10% padding) and anchors it with a natural grounding offset.
6. Encodes the final composed scene to high-quality JPEG or PNG.

---

## 2. Architecture & Design Principles
- **Pure Go**: 100% standard library `image`, `image/color`, `image/draw`, `image/jpeg`, `image/png`. Zero cgo. Zero external C libraries (no ImageMagick, no libvips).
- **Preset Catalog Themes**:
  - `ThemeCosmetics`: Luxury soft pearl/glass background (`#F6F8FB`) + crisp ground reflection shadow.
  - `ThemeEthnicFashion`: Warm aesthetic gallery canvas (`#FAF9F6`) + soft diffused ambient floor shadow.
  - `ThemePackagedRetail`: Crisp pure catalog white (`#FFFFFF`) + tight base contact shadow.
  - `ThemeMinimalistDark`: Obsidian charcoal dark mode (`#18181B`).
- **Category-Aware Inference**: `DetermineStudioTheme` infers the optimal theme automatically from product metadata.

---

## 3. Public API Specification
```go
package studiocomp

type Theme string

const (
    ThemeCosmetics      Theme = "cosmetics"
    ThemeEthnicFashion  Theme = "ethnic_fashion"
    ThemePackagedRetail Theme = "packaged_retail"
    ThemeMinimalistDark Theme = "minimalist_dark"
)

type Options struct {
    Theme        Theme
    CanvasSize   int     // Default 1080
    PaddingRatio float64 // Default 0.10
    JpegQuality  int     // Default 90
}

func DetermineStudioTheme(categoryHint, subcategory string) Theme
func RenderProductStudio(cutoutBytes []byte, opts Options) ([]byte, error)
func FindOpaqueBounds(src image.Image) (minX, minY, maxX, maxY int, hasAlpha bool)
func BilinearScale(dst *image.RGBA, src image.Image)
func FeatherEdges(img *image.RGBA)
func FillCanvasBackdrop(canvas *image.RGBA, theme Theme)
func DrawContactShadow(canvas *image.RGBA, theme Theme, posX, posY, pWidth, pHeight int)
```

---

## 4. Performance Targets
- Standard 512x512 studio render: < 35 ms
- 1080x1080 standard e-commerce composite: < 80 ms
- Memory allocation: Minimal buffers, in-place alpha feathering
