// Package exif provides functionality to extract EXIF metadata from images.
package exif

import (
	"brd-shapify/internal/core/domain"
	"bytes"

	"github.com/rwcarlsen/goexif/exif"
)

type ExifAdapter struct{}

func NewExifAdapter() *ExifAdapter {
	return &ExifAdapter{}
}

func (a *ExifAdapter) GetOrientation(data []byte) (int, error) {
	exifData, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return domain.OrientationNormal, nil
	}

	orient, err := exifData.Get(exif.Orientation)
	if err != nil {
		return domain.OrientationNormal, nil
	}

	orientVal, _ := orient.Int(0)
	return orientVal, nil
}

func (a *ExifAdapter) GetMetadata(data []byte) (*domain.ExifData, error) {
	exifData, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil
	}

	result := &domain.ExifData{}

	if orient, err := exifData.Get(exif.Orientation); err == nil {
		orientVal, _ := orient.Int(0)
		result.Orientation = orientVal
	}

	if make, err := exifData.Get(exif.Make); err == nil {
		result.Make = make.String()
	}

	if model, err := exifData.Get(exif.Model); err == nil {
		result.Model = model.String()
	}

	if date, err := exifData.Get(exif.DateTime); err == nil {
		result.DateTime = date.String()
	}

	if soft, err := exifData.Get(exif.Software); err == nil {
		result.Software = soft.String()
	}

	return result, nil
}
