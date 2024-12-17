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
	"log/slog"
)

type Writer struct {
	fsys WriterFS
}

func NewWriter(fsys WriterFS) *Writer {
	return &Writer{
		fsys: fsys,
	}
}

func (wrt Writer) Put(ctx context.Context, media *Media) (err error) {
	slog.Debug("write media object",
		slog.String("path", media.path),
		slog.Group("source", "x", media.image.Bounds().Dx(), "y", media.image.Bounds().Dy()),
	)

	path := media.path + ".jpg"
	fd, err := wrt.fsys.Create(path, &Meta{ContentType: "image/jpg"})
	if err != nil {
		return errCodecIO.New(err)
	}
	defer func() { err = fd.Close() }()

	// TODO: Make customizable but 93% is optimal
	if err := jpeg.Encode(fd, media.image, &jpeg.Options{Quality: 93}); err != nil {
		slog.Error("failed encode jpeg", "error", err)
	}

	return nil
}
