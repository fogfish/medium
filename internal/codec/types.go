//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package codec

import (
	"image"
	"io/fs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/fogfish/faults"
	"github.com/fogfish/stream"
)

// Abstract media file reader.
type ReaderFS = fs.FS

// Abstract media file writer.
type WriterFS = stream.CreateFS[Meta]

type Meta struct {
	ContentType string
}

// Event emitted (TODO: fix naming)
type Event struct {
	events.S3EventRecord
	Variants []string
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
	path  string
	image image.Image
}

type Link struct {
	Url string `json:"url"`
}
