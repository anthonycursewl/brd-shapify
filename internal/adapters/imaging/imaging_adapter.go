// Package imaging handles image processing operations.
package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strconv"
	"strings"

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

func loadFile(_ string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented: use injected dependency")
}

func (a *ImageProcessorAdapter) Process(img image.Image, opts domain.ProcessOptions) ([]byte, error) {
	processed := img

	if opts.Blur > 0 {
		processed = a.Blur(processed, opts.Blur)
	}

	if opts.Sharpen > 0 {
		processed = a.Sharpen(processed, opts.Sharpen)
	}

	if opts.Brightness != 0 {
		processed = a.AdjustBrightness(processed, opts.Brightness)
	}

	if opts.Contrast != 0 {
		processed = a.AdjustContrast(processed, opts.Contrast)
	}

	if opts.Saturation != 0 {
		processed = a.AdjustSaturation(processed, opts.Saturation)
	}

	if opts.Grayscale {
		processed = a.Grayscale(processed)
	}

	if opts.Width > 0 || opts.Height > 0 {
		var err error
		processed, err = a.processResize(processed, opts.Width, opts.Height, opts.Fit)
		if err != nil {
			return nil, fmt.Errorf("resize failed: %w", err)
		}
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
	if quality < 1 || quality > 100 {
		return nil, fmt.Errorf("quality must be between 1 and 100, got %d", quality)
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

func (a *ImageProcessorAdapter) Grayscale(img image.Image) image.Image {
	return imaging.Grayscale(img)
}

func (a *ImageProcessorAdapter) Blur(img image.Image, sigma float64) image.Image {
	return imaging.Blur(img, sigma)
}

func (a *ImageProcessorAdapter) Sharpen(img image.Image, sigma float64) image.Image {
	return imaging.Sharpen(img, sigma)
}

func (a *ImageProcessorAdapter) AdjustBrightness(img image.Image, brightness float64) image.Image {
	return imaging.AdjustBrightness(img, brightness)
}

func (a *ImageProcessorAdapter) AdjustContrast(img image.Image, contrast float64) image.Image {
	return imaging.AdjustContrast(img, contrast)
}

func (a *ImageProcessorAdapter) AdjustSaturation(img image.Image, saturation float64) image.Image {
	return imaging.AdjustSaturation(img, saturation)
}

type Color struct {
	Hex  string `json:"hex"`
	R    uint8   `json:"r"`
	G    uint8   `json:"g"`
	B    uint8   `json:"b"`
	Count int    `json:"count"`
}

func (a *ImageProcessorAdapter) ExtractPalette(img image.Image, numColors int) ([]Color, error) {
	if numColors < 1 {
		numColors = 5
	}
	if numColors > 20 {
		numColors = 20
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	colorCounts := make(map[string]int)
	var r, g, b uint8

	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 2 {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			r = uint8(cr >> 8)
			g = uint8(cg >> 8)
			b = uint8(cb >> 8)

			quantR := (r / 32) * 32
			quantG := (g / 32) * 32
			quantB := (b / 32) * 32

			key := fmt.Sprintf("%d,%d,%d", quantR, quantG, quantB)
			colorCounts[key]++
		}
	}

	type colorCount struct {
		key   string
		count int
	}
	var sorted []colorCount
	for k, v := range colorCounts {
		sorted = append(sorted, colorCount{k, v})
	}

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := make([]Color, 0, numColors)
	for i := 0; i < len(sorted) && i < numColors; i++ {
		parts := strings.Split(sorted[i].key, ",")
		if len(parts) != 3 {
			continue
		}
		ri, _ := strconv.ParseUint(parts[0], 10, 8)
		gi, _ := strconv.ParseUint(parts[1], 10, 8)
		bi, _ := strconv.ParseUint(parts[2], 10, 8)

		hex := fmt.Sprintf("#%02X%02X%02X", uint8(ri), uint8(gi), uint8(bi))
		result = append(result, Color{
			Hex:  hex,
			R:    uint8(ri),
			G:    uint8(gi),
			B:    uint8(bi),
			Count: sorted[i].count,
		})
	}

	return result, nil
}