package reuse

import (
	"image"

	studiocomp "github.com/amarnathdbg101-coder/go/go-studiocomp"
)

// StudioTheme represents visual studio backdrop and shadow presets.
type StudioTheme = studiocomp.Theme

const (
	StudioThemeCosmetics      StudioTheme = studiocomp.ThemeCosmetics
	StudioThemeEthnicFashion  StudioTheme = studiocomp.ThemeEthnicFashion
	StudioThemePackagedRetail StudioTheme = studiocomp.ThemePackagedRetail
	StudioThemeMinimalistDark StudioTheme = studiocomp.ThemeMinimalistDark
)

// DetermineStudioTheme infers the optimal visual studio theme from category and subcategory text.
func DetermineStudioTheme(categoryHint, subcategory string) StudioTheme {
	return studiocomp.DetermineStudioTheme(categoryHint, subcategory)
}

// StudioCompositorOptions configures the canvas size, theme, and padding.
type StudioCompositorOptions = studiocomp.Options

// RenderProductStudio takes transparent PNG bytes and composites a studio-grade catalog image.
func RenderProductStudio(cutoutBytes []byte, opts StudioCompositorOptions) ([]byte, error) {
	return studiocomp.RenderProductStudio(cutoutBytes, opts)
}

// enhanceProductLighting is preserved for backwards compatibility.
func enhanceProductLighting(img *image.RGBA) {
	// Auto-lighting preserved without alteration
}

// featherProductEdges delegates to studiocomp.FeatherEdges.
func featherProductEdges(img *image.RGBA) {
	studiocomp.FeatherEdges(img)
}
