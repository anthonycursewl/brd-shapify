package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
)

type ImageProcessorAdapter struct{}

func NewImageProcessorAdapter() *ImageProcessorAdapter {
	return &ImageProcessorAdapter{}
}

func (a *ImageProcessorAdapter) Resize(img image.Image, width, height int) (image.Image, error) {
	return imaging.Fill(img, width, height, imaging.Center, imaging.Box), nil
}

func (a *ImageProcessorAdapter) Compress(img image.Image, quality int) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *ImageProcessorAdapter) Convert(img image.Image, format string) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error

	switch format {
	case "jpg", "jpeg":
		err = jpeg.Encode(buf, img, nil)
	case "png":
		err = png.Encode(buf, img)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
