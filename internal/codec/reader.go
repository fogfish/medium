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
	"encoding/json"
	"image"
	_ "image/jpeg"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/fogfish/gurl/v2/http"
	ƒ "github.com/fogfish/gurl/v2/http/recv"
	ø "github.com/fogfish/gurl/v2/http/send"
	"github.com/fogfish/medium"
	"github.com/fogfish/swarm/broker/events3"
)

type Reader struct {
	http.Stack
	getter Getter
}

func NewReader(stack http.Stack, getter Getter) *Reader {
	return &Reader{
		Stack:  stack,
		getter: getter,
	}
}

func (r Reader) Get(ctx context.Context, evt *events3.Event) (*Media, error) {
	format, supported := r.isSupported(evt.Object.S3.Object.Key)
	if !supported {
		return nil, errCodecNotSupported.New(nil, format)
	}

	key, err := medium.NewMediaFromPath(evt.Object.S3.Object.Key)
	if err != nil {
		return nil, errCodecIO.New(err)
	}

	slog.Debug("getting media object",
		slog.String("bucket", evt.Object.S3.Bucket.Name),
		slog.String("key", evt.Object.S3.Object.Key),
		"media", key,
	)

	switch format {
	case MEDIA_JPEG:
		return r.fetchMediaJpeg(ctx, key)
	case MEDIA_LINK:
		return r.fetchMediaLink(ctx, key)
	}

	return nil, errCodecNotSupported.New(nil, format)
}

func (r Reader) isSupported(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg":
		return MEDIA_JPEG, true
	case ".json":
		return MEDIA_LINK, true
	default:
		return ext, false
	}
}

func (r Reader) fetchMediaJpeg(ctx context.Context, key *medium.Media) (*Media, error) {
	_, stream, err := r.getter.Get(ctx, key)
	if err != nil {
		return nil, errCodecIO.New(err)
	}

	img, _, err := image.Decode(stream)
	if err != nil {
		return nil, errCodecIO.New(err)
	}

	return &Media{
		key:   key,
		image: img,
	}, nil
}

func (r Reader) fetchMediaLink(ctx context.Context, key *medium.Media) (*Media, error) {
	_, stream, err := r.getter.Get(ctx, key)
	if err != nil {
		return nil, errCodecIO.New(err)
	}

	var link Link
	if err := json.NewDecoder(stream).Decode(&link); err != nil {
		return nil, err
	}

	img, err := r.fetchMediaFile(ctx, link.Url)
	if err != nil {
		return nil, errCodecIO.New(err)
	}

	return &Media{
		key:   key,
		image: *img,
	}, nil
}

func (r Reader) fetchMediaFile(ctx context.Context, url string) (*image.Image, error) {
	return http.IO[image.Image](r.WithContext(ctx),
		http.GET(
			ø.URI(url),
			ø.Accept.Set("image/*"),
			ƒ.Status.OK,
		),
	)
}
