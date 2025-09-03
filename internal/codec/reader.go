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
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"

	"log/slog"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/fogfish/gurl/v2/http"
	ƒ "github.com/fogfish/gurl/v2/http/recv"
	ø "github.com/fogfish/gurl/v2/http/send"
	"github.com/fogfish/swarm"
)

type Reader struct {
	http.Stack
	fsys ReaderFS
}

func NewReader(stack http.Stack, fsys ReaderFS) *Reader {
	return &Reader{
		Stack: stack,
		fsys:  fsys,
	}
}

func (r Reader) Get(ctx context.Context, evt swarm.Msg[*events.S3EventRecord]) (*Media, error) {
	path, err := url.QueryUnescape(evt.Object.S3.Object.Key)
	if err != nil {
		return nil, err
	}

	format, supported := r.isSupported(path)
	if !supported {
		return nil, errCodecNotSupported.With(nil, format)
	}

	slog.Debug("getting media object",
		slog.String("bucket", evt.Object.S3.Bucket.Name),
		slog.String("key", evt.Object.S3.Object.Key),
	)

	path = filepath.Join("/", path)
	switch format {
	case MEDIA_JPEG:
		return r.fetchMediaJpeg(ctx, path)
	case MEDIA_LINK:
		return r.fetchMediaLink(ctx, path)
	}

	return nil, errCodecNotSupported.With(nil, format)
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

func (r Reader) fetchMediaJpeg(_ context.Context, path string) (*Media, error) {
	fd, err := r.fsys.Open(path)
	if err != nil {
		return nil, errCodecIO.With(err)
	}
	defer fd.Close()

	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, errCodecIO.With(err)
	}

	return &Media{
		path:  path,
		image: img,
	}, nil
}

func (r Reader) fetchMediaLink(ctx context.Context, path string) (*Media, error) {
	fd, err := r.fsys.Open(path)
	if err != nil {
		return nil, errCodecIO.With(err)
	}
	defer fd.Close()

	var link Link
	if err := json.NewDecoder(fd).Decode(&link); err != nil {
		return nil, err
	}

	img, err := r.fetchMediaFile(ctx, link.Url)
	if err != nil {
		return nil, errCodecIO.With(err)
	}

	return &Media{
		path:  path,
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
