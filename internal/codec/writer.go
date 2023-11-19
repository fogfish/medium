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
	"image/jpeg"
	"io"
	"log/slog"
)

type Writer struct {
	putter Putter
}

func NewWriter(putter Putter) *Writer {
	return &Writer{
		putter: putter,
	}
}

func (wrt Writer) Put(ctx context.Context, media *Media) error {
	slog.Debug("write media object",
		slog.String("key", media.key.PathKey()),
		slog.Group("source", "x", media.image.Bounds().Dx(), "y", media.image.Bounds().Dy()),
	)

	r, w := io.Pipe()

	// TODO: Make customizable but 93% is optimal
	go func() {
		defer w.Close()
		if err := jpeg.Encode(w, media.image, &jpeg.Options{Quality: 93}); err != nil {
			slog.Error("failed encode jpeg", "error", err)
		}
	}()

	media.key.SortID = media.key.SortID + ".jpg"
	media.key.ContentType = "image/jpg"

	if err := wrt.putter.Put(ctx, media.key, r); err != nil {
		return errCodecIO.New(err)
	}

	return nil
}
