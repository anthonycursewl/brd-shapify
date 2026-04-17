// Package imaging handles image processing operations.
package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"brd-shapify/internal/core/domain"

	"github.com/disintegration/imaging"
	"github.com/skrashevich/go-webp"
)

type ImageProcessorAdapter struct {
	watermarkPath string
	watermarkImg  image.Image
}

func NewImageProcessorAdapter(watermarkPath string) (*ImageProcessorAdapter, error) {
	adapter := &ImageProcessorAdapter{watermarkPath: watermarkPath}
	if watermarkPath != "" {
		data, err := loadWatermark(watermarkPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load watermark: %w", err)
		}
		adapter.watermarkImg = data
	}
	return adapter, nil
}

func loadWatermark(path string) (image.Image, error) {
	data, err := loadFile(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func loadFile(path string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented: use injected dependency")
}

func (a *ImageProcessorAdapter) Process(img image.Image, opts domain.ProcessOptions) ([]byte, error) {
	var processed image.Image = img

	if opts.Width > 0 || opts.Height > 0 {
		processed, err := a.processResize(processed, opts.Width, opts.Height, opts.Fit)
		if err != nil {
			return nil, fmt.Errorf("resize failed: %w", err)
		}
		processed = processed
	}

	if opts.Watermark != nil && opts.Watermark.Enabled && a.watermarkImg != nil {
		var err error
		processed, err = a.Watermark(processed, *opts.Watermark)
		if err != nil {
			return nil, fmt.Errorf("watermark failed: %w", err)
		}
	}

	format := domain.ImageFormat(opts.Format)
	switch format {
	case domain.FormatWebP:
		quality := opts.Quality
		if quality <= 0 {
			quality = 85
		}
		return a.EncodeWebP(processed, quality)
	case domain.FormatJPEG:
		quality := opts.Quality
		if quality <= 0 {
			quality = 85
		}
		return a.Compress(processed, quality)
	case domain.FormatPNG:
		return a.Convert(processed, "png")
	default:
		quality := opts.Quality
		if quality <= 0 {
			quality = 85
		}
		return a.Compress(processed, quality)
	}
}

func (a *ImageProcessorAdapter) processResize(img image.Image, width, height int, fit string) (image.Image, error) {
	if fit == "fit" {
		return imaging.Fit(img, width, height, imaging.Lanczos), nil
	}
	if fit == "scale" {
		return imaging.Resize(img, width, height, imaging.Lanczos), nil
	}
	if fit == "thumb" {
		return imaging.Thumbnail(img, width, height, imaging.Lanczos), nil
	}
	return imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos), nil
}

func (a *ImageProcessorAdapter) AutoRotate(img image.Image, orientation int) image.Image {
	switch orientation {
	case domain.OrientationMirrorHorizontal:
		return imaging.FlipH(img)
	case domain.OrientationRotate180:
		return imaging.Rotate180(img)
	case domain.OrientationMirrorVertical:
		return imaging.FlipV(img)
	case domain.OrientationMirrorTopLeft:
		return imaging.Rotate180(imaging.FlipH(img))
	case domain.OrientationRotate90Left:
		return imaging.Rotate270(img)
	case domain.OrientationMirrorTopRight:
		return imaging.Rotate90(imaging.FlipV(img))
	case domain.OrientationRotate90Right:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

func (a *ImageProcessorAdapter) Resize(img image.Image, width, height int) (image.Image, error) {
	return a.processResize(img, width, height, "fill")
}

func (a *ImageProcessorAdapter) Compress(img image.Image, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = 85
	}
	if quality > 100 {
		quality = 100
	}
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
	case "webp":
		return nil, fmt.Errorf("webp not supported in basic convert, use Process with format option")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *ImageProcessorAdapter) Watermark(img image.Image, cfg domain.WatermarkConfig) (image.Image, error) {
	if a.watermarkImg == nil {
		return img, nil
	}

	wmBounds := a.watermarkImg.Bounds()
	imgBounds := img.Bounds()

	var x, y int
	switch cfg.Preset {
	case "center":
		x = (imgBounds.Dx() - wmBounds.Dx()) / 2
		y = (imgBounds.Dy() - wmBounds.Dy()) / 2
	case "top-left":
		x = cfg.OffsetX
		y = cfg.OffsetY
	case "top-right":
		x = imgBounds.Dx() - wmBounds.Dx() - cfg.OffsetX
		y = cfg.OffsetY
	case "bottom-left":
		x = cfg.OffsetX
		y = imgBounds.Dy() - wmBounds.Dy() - cfg.OffsetY
	default:
		x = imgBounds.Dx() - wmBounds.Dx() - cfg.OffsetX
		y = imgBounds.Dy() - wmBounds.Dy() - cfg.OffsetY
	}

	wmResized := imaging.Resize(a.watermarkImg, wmBounds.Dx(), wmBounds.Dy(), imaging.Lanczos)

	return imaging.Overlay(img, wmResized, image.Pt(x, y), 255), nil
}

func adjustOpacity(img image.Image, opacity float32) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			rgba.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(float32(a>>8) * opacity),
			})
		}
	}
	return rgba
}

func (a *ImageProcessorAdapter) EncodeWebP(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := webp.Encode(&buf, img, &webp.Options{
		Lossy:   true,
		Quality: float32(quality),
	})
	if err != nil {
		return nil, fmt.Errorf("webp encoding failed: %w", err)
	}
	return buf.Bytes(), nil
}