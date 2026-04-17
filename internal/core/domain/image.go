package domain

import "image"

type ProcessOptions struct {
	Width    int
	Height   int
	Format   string
	Quality  int
	Fit      string
	Watermark *WatermarkConfig
}

type WatermarkConfig struct {
	Enabled  bool
	Preset   string
	OffsetX  int
	OffsetY  int
	Opacity  float32
}

type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatWebP ImageFormat = "webp"
)

type FitMode string

const (
	FitFill   FitMode = "fill"
	FitFit    FitMode = "fit"
	FitScale  FitMode = "scale"
	FitThumb  FitMode = "thumb"
)

type ImageMetadata struct {
	Width       int
	Height      int
	Format      string
	Orientation int
	HasEXIF     bool
}

func (o *ProcessOptions) ToImageConfig() image.Config {
	return image.Config{
		Width:  o.Width,
		Height: o.Height,
	}
}

func (o *ProcessOptions) Validate() error {
	if o.Quality < 0 || o.Quality > 100 {
		return ErrInvalidQuality
	}
	return nil
}

var ErrInvalidQuality = &DomainError{"quality must be between 1 and 100"}

type DomainError struct {
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

type ExifData struct {
	Orientation  int
	Make         string
	Model        string
	DateTime     string
	Software     string
	GPSLatitude  float64
	GPSLongitude float64
}

const (
	OrientationNormal           = 1
	OrientationMirrorHorizontal = 2
	OrientationRotate180       = 3
	OrientationMirrorVertical  = 4
	OrientationMirrorTopLeft   = 5
	OrientationRotate90Left    = 6
	OrientationMirrorTopRight  = 7
	OrientationRotate90Right   = 8
)
