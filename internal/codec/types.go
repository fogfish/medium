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
	"io"

	"github.com/fogfish/faults"
	"github.com/fogfish/medium"
)

// Abstract media file reader, provide interface implementation.
type Getter interface {
	Get(context.Context, *medium.Media, ...interface{ GetterOpt(*medium.Media) }) (*medium.Media, io.ReadCloser, error)
}

// Abstract media file writer, provide interface implementation.
type Putter interface {
	Put(context.Context, *medium.Media, io.Reader, ...interface{ WriterOpt(*medium.Media) }) error
}

const (
	errCodecIO           = faults.Type("codec I/O error")
	errCodecNotSupported = faults.Safe1[string]("not supported (%s)")
)

const (
	MEDIA_JPEG = "jpeg"
	MEDIA_LINK = "link"
)

// Container for digital media
type Media struct {
	key   *medium.Media
	image image.Image
}

type Link struct {
	Url string `json:"url"`
}
