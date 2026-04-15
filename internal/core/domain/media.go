package domain

import "image"

type MediaFile struct {
	ID       string
	Data     image.Image
	Format   string
	Width    int
	Height   int
}

type ImageProcessingRequest struct {
	Width       int
	Height      int
	Compression int   
	Format      string
}
