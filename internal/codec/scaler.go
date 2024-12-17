//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package codec

import (
	"context"
	"image"
	"log/slog"

	"github.com/anthonynsimon/bild/transform"
	"github.com/fogfish/medium"
)

type Scaler struct {
	resolution medium.Resolution
}

func NewScaler(resolution medium.Resolution) *Scaler {
	return &Scaler{
		resolution: resolution,
	}
}

func (s Scaler) Process(ctx context.Context, media *Media) (*Media, error) {
	slog.Debug("scaling media object",
		slog.String("path", media.path),
		slog.Group("source", "x", media.image.Bounds().Dx(), "y", media.image.Bounds().Dy()),
		slog.Group("target", "x", s.resolution.Width, "y", s.resolution.Height),
	)

	if s.resolution.Width == 0 || s.resolution.Height == 0 {
		return s.replica(ctx, media)
	}

	return s.scaleTo(ctx, media)
}

func (s Scaler) replica(_ context.Context, media *Media) (*Media, error) {
	return &Media{
		path:  s.resolution.FileSuffix(media.path),
		image: media.image,
	}, nil
}

func (s Scaler) scaleTo(_ context.Context, media *Media) (*Media, error) {
	cropX, cropY := CropToScale(
		image.Point{
			X: media.image.Bounds().Dx(),
			Y: media.image.Bounds().Dy(),
		},
		image.Point{
			X: s.resolution.Width,
			Y: s.resolution.Height,
		},
	)

	cropped := transform.Crop(media.image,
		image.Rect(
			cropX/2,
			cropY/2,
			media.image.Bounds().Dx()-cropX/2,
			media.image.Bounds().Dy()-cropY/2,
		),
	)

	img := transform.Resize(cropped, s.resolution.Width, s.resolution.Height, transform.Lanczos)

	return &Media{
		path:  s.resolution.FileSuffix(media.path),
		image: img,
	}, nil
}

// CropToScale calculates a new dimension of image
func CropToScale(source image.Point, target image.Point) (int, int) {
	aspectSource := float64(source.X) / float64(source.Y)
	aspectTarget := float64(target.X) / float64(target.Y)

	if aspectSource > aspectTarget {
		width := int(float64(source.Y) * aspectTarget)
		return source.X - width, 0
	}

	if aspectSource < aspectTarget {
		height := int(float64(source.X) / aspectTarget)
		return 0, source.Y - height
	}

	return 0, 0
}
