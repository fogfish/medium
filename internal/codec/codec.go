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

	"github.com/fogfish/gurl/v2/http"
	"github.com/fogfish/medium"
	"github.com/fogfish/swarm/broker/events3"
	"golang.org/x/sync/errgroup"
)

type Codec struct {
	reader *Reader
	scaler []*Scaler
	writer *Writer
}

func NewCodec(profile medium.Profile, getter Getter, putter Putter) *Codec {
	// defines HTTP client to download media objects
	client := http.Client()
	client.CheckRedirect = nil
	stack := http.New(http.WithClient(client))

	reader := NewReader(stack, getter)

	scaler := make([]*Scaler, len(profile.Resolutions))
	for i, r := range profile.Resolutions {
		scaler[i] = NewScaler(r)
	}

	writer := NewWriter(putter)

	return &Codec{
		reader: reader,
		scaler: scaler,
		writer: writer,
	}
}

func (codec *Codec) Process(ctx context.Context, evt *events3.Event) error {
	media, err := codec.reader.Get(ctx, evt)
	if err != nil {
		return errCodecIO.New(err)
	}

	var g errgroup.Group

	for _, scaler := range codec.scaler {
		s := scaler

		g.Go(func() error {
			img, err := s.Process(ctx, media)
			if err != nil {
				return err
			}

			return codec.writer.Put(ctx, img)
		})
	}

	if err := g.Wait(); err != nil {
		return errCodecIO.New(err)
	}

	return nil
}
