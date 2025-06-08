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

	"github.com/aws/aws-lambda-go/events"
	_ "github.com/fogfish/logger/v3"
	"github.com/fogfish/medium"
	"github.com/fogfish/medium/internal/codec"
	"github.com/fogfish/stream"
	"github.com/fogfish/swarm"
	"github.com/fogfish/swarm/broker/eventbridge"
	"github.com/fogfish/swarm/broker/events3"
	"github.com/fogfish/swarm/enqueue"
)

func Runner() {
	q, err := events3.NewDequeuer(
		events3.WithConfig(
			swarm.WithLogStdErr(),
			swarm.WithTimeToFlight(60*time.Second),
			swarm.WithConfigFromEnv(),
		),
	)
	if err != nil {
		slog.Error("Failed to init events3 broker")
		panic(err)
	}

	inbox, err := stream.NewFS(os.Getenv("CONFIG_STORE_INBOX"))
	if err != nil {
		slog.Error("Failed to init inbox s3 client")
		panic(err)
	}

	media, err := stream.New[codec.Meta](os.Getenv("CONFIG_STORE_MEDIA"))
	if err != nil {
		slog.Error("Failed to init media s3 client")
		panic(err)
	}

	profile, err := medium.NewProfile(os.Getenv("CONFIG_CODEC_PROFILE"))
	if err != nil {
		slog.Error("Invalid profile", "profile", os.Getenv("CONFIG_CODEC_PROFILE"))
		panic(err)
	}

	var emitter codec.Emitter
	eventbus := os.Getenv("CONFIG_SINK_EVENTBUS")
	if eventbus != "" {
		bridge, err := eventbridge.NewEnqueuer(
			eventbridge.WithEventBus(eventbus),
			eventbridge.WithConfig(
				swarm.WithSource("github.com/fogfish/medium"),
				swarm.WithLogStdErr(),
			),
		)
		if err != nil {
			slog.Error("Failed to init eventbridge broker", "bus", eventbus)
			panic(err)
		}
		emitter = enqueue.NewTyped[codec.Event](bridge)
	}

	codec := codec.NewCodec(profile, inbox, media, emitter)

	bus := bus{codec: codec}
	go bus.onEventS3(events3.Source(q))

	q.Await()
}

type bus struct {
	codec interface {
		Process(context.Context, swarm.Msg[*events.S3EventRecord]) error
	}
}

func (bus *bus) onEventS3(rcv <-chan swarm.Msg[*events.S3EventRecord], ack chan<- swarm.Msg[*events.S3EventRecord]) {
	for evt := range rcv {
		err := bus.codec.Process(context.Background(), evt)
		if err != nil {
			slog.Error("failed to process s3 event",
				slog.String("bucket", evt.Object.S3.Bucket.Name),
				slog.String("key", evt.Object.S3.Object.Key),
				"error", err,
			)
			ack <- evt.Fail(err)
			continue
		}

		ack <- evt
	}
}
