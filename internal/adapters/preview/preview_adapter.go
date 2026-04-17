package preview

import (
	"image"

	"github.com/bbrks/go-blurhash"
)

type PreviewAdapter struct{}

func NewPreviewAdapter() *PreviewAdapter {
	return &PreviewAdapter{}
}

func (a *PreviewAdapter) GenerateBlurHash(img image.Image) (string, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return "", nil
	}

	componentsX := 4
	componentsY := 4
	if width < 4 {
		componentsX = width
	}
	if height < 4 {
		componentsY = height
	}

	hash, err := blurhash.Encode(componentsX, componentsY, img)
	if err != nil {
		return "", err
	}

	return hash, nil
}