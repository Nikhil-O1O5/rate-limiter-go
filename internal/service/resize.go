package service

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"golang.org/x/image/draw"
)

type ResizeService struct{}

func NewResizeService() *ResizeService {
	return &ResizeService{}
}

func (s *ResizeService) Resize(data []byte, width, height int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	return buf.Bytes(), nil
}
