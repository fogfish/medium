//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package inbox

import (
	"context"
	"log/slog"
	"os"
	"time"

	_ "github.com/fogfish/logger/v3"
	"github.com/fogfish/medium"
	"github.com/fogfish/medium/internal/codec"
	"github.com/fogfish/stream/service/s3"
	"github.com/fogfish/swarm"
	"github.com/fogfish/swarm/broker/events3"
)

func Runner() {
	q, err := events3.New(
		os.Getenv("CONFIG_STORE_INBOX"),
		swarm.WithLogStdErr(),
		swarm.WithTimeToFlight(60*time.Second),
		swarm.WithConfigFromEnv(),
	)
	if err != nil {
		slog.Error("Failed to init events3 broker")
		panic(err)
	}

	inbox, err := s3.New[*medium.Media](
		s3.WithBucket(os.Getenv("CONFIG_STORE_INBOX")),
	)
	if err != nil {
		slog.Error("Failed to init inbox s3 client")
		panic(err)
	}

	media, err := s3.New[*medium.Media](
		s3.WithBucket(os.Getenv("CONFIG_STORE_MEDIA")),
	)
	if err != nil {
		slog.Error("Failed to init media s3 client")
		panic(err)
	}

	profile, err := medium.NewProfile(os.Getenv("CONFIG_CODEC_PROFILE"))
	if err != nil {
		slog.Error("Invalid profile", "profile", os.Getenv("CONFIG_CODEC_PROFILE"))
		panic(err)
	}

	bus := bus{
		codec: codec.NewCodec(profile, inbox, media),
	}

	go bus.onEventS3(events3.Dequeue(q))

	q.Await()
}

type bus struct {
	codec interface {
		Process(context.Context, *events3.Event) error
	}
}

func (bus *bus) onEventS3(rcv <-chan *events3.Event, ack chan<- *events3.Event) {
	for evt := range rcv {
		evt.Digest.Error = bus.codec.Process(context.Background(), evt)

		if evt.Digest.Error != nil {
			slog.Error("failed to process s3 event",
				slog.String("bucket", evt.Object.S3.Bucket.Name),
				slog.String("key", evt.Object.S3.Object.Key),
				"error", evt.Digest.Error,
			)
		}

		ack <- evt
	}
}
