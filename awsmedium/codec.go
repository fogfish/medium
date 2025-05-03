//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package awsmedium

import (
	"path/filepath"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/medium"
	"github.com/fogfish/scud"
	"github.com/fogfish/swarm/broker/events3"
	"github.com/fogfish/tagver"

	// Note: required to import engine so that all deps used it are lifted to client.
	//       app that uses only stack fails to build if image manipulation library is not imported.
	//       e.g. github.com/anthonynsimon/bild
	_ "github.com/fogfish/medium/internal/awslambda/inbox"
)

type CodecProps struct {
	*awscdk.StackProps

	// Namespace for cloud resource provisioning
	Namespace string

	// Version of the deployment
	Version tagver.Version

	// Configuration profiles (mandatory).
	// Each profile defines:
	// - the prefix of S3 Key identify the profile
	// - processing stages that produces media files from the original one.
	//
	//  medium.Profiles(
	//    medium.On("photo").Process(
	//      medium.ScaleTo("small", 128, 128),
	//      medium.ScaleTo("thumb", 240, 240),
	//      medium.ScaleTo("cover", 480, 720),
	//      medium.ScaleTo("large", 1080, 1920),
	//      medium.Replica("origin"),
	//    )
	//  )
	//
	Profiles []medium.Profile

	// The amount of memory, in MB, that is allocated to your Lambda function.
	//
	// Lambda uses this value to proportionally allocate the amount of CPU
	// power. For more information, see Resource Model in the AWS Lambda
	// Developer Guide.
	// Default: 128.
	//
	MemorySize *float64

	// Deadline for running processing pipeline.
	// The processing pipelines are terminated with force, the result is not predictable.
	// Default: 60 seconds
	//
	Deadline awscdk.Duration

	// Retention of failed operations in Dead-Letter Queue
	// Default: 1 day
	//
	FailureRetention awscdk.Duration

	// Expiration of content in the inbox
	// Default: 1 day
	//
	Expiration awscdk.Duration

	// EventBus to emit event upon the completion
	// Default: None
	//
	EventBus awsevents.IEventBus

	// AWS S3 bucket to write media files
	Media awss3.IBucket
}

func (props *CodecProps) assert() {
	if len(props.Profiles) == 0 {
		panic("\n\nMedia processing profiles are not defined.")
	}

	if props.Deadline == nil {
		props.Deadline = awscdk.Duration_Seconds(jsii.Number(60.0))
	}

	if props.FailureRetention == nil {
		props.FailureRetention = awscdk.Duration_Days(jsii.Number(1.0))
	}

	if props.Expiration == nil {
		props.Expiration = awscdk.Duration_Days(jsii.Number(1.0))
	}
}

type Codec struct {
	awscdk.Stack
	namespace string
	version   tagver.Version

	dlq   awssqs.Queue
	Inbox awss3.Bucket
}

func NewCodec(app awscdk.App, id *string, props *CodecProps) *Codec {
	props.assert()

	stack := &Codec{
		Stack:     awscdk.NewStack(app, id, props.StackProps),
		namespace: props.Namespace,
		version:   props.Version,
	}

	stack.createDLQ(props)
	stack.createInboxBucket(props)
	for _, profile := range props.Profiles {
		stack.createInboxCodec(props, profile)
	}

	return stack
}

func (stack *Codec) resource(id string) string {
	return stack.version.Tag(stack.namespace + "-" + id)
}

func (stack *Codec) createDLQ(props *CodecProps) {
	name := stack.resource("dlq")

	stack.dlq = awssqs.NewQueue(stack.Stack, jsii.String("DeadLetterQueue"),
		&awssqs.QueueProps{
			QueueName:       jsii.String(name),
			RetentionPeriod: props.FailureRetention,
		},
	)
}

func (stack *Codec) createInboxBucket(props *CodecProps) {
	name := stack.resource("inbox")

	policy := awscdk.RemovalPolicy_RETAIN
	if tagver.IsTest(props.Version) {
		policy = awscdk.RemovalPolicy_DESTROY
	}

	stack.Inbox = awss3.NewBucket(stack.Stack, jsii.String("Inbox"),
		&awss3.BucketProps{
			BucketName:    jsii.String(name),
			RemovalPolicy: policy,
		},
	)

	stack.Inbox.AddLifecycleRule(&awss3.LifecycleRule{
		Id:         jsii.String("Garbage collector"),
		Enabled:    jsii.Bool(true),
		Expiration: props.Expiration,
	})
}

func (stack *Codec) createInboxCodec(props *CodecProps, profile medium.Profile) {
	path := profile.Path
	name := stack.resource("inbox-codec-" + filepath.Base(path))
	tout := props.Deadline.ToSeconds(&awscdk.TimeConversionOptions{})

	envs := map[string]*string{
		"CONFIG_STORE_INBOX":          stack.Inbox.BucketName(),
		"CONFIG_STORE_MEDIA":          props.Media.BucketName(),
		"CONFIG_CODEC_PROFILE":        jsii.String(profile.String()),
		"CONFIG_SWARM_TIME_TO_FLIGHT": jsii.String(strconv.Itoa(int(*tout))),
	}
	if props.EventBus != nil {
		envs["CONFIG_SINK_EVENTBUS"] = props.EventBus.EventBusName()
	}

	sink := events3.NewSink(stack.Stack, jsii.String("InboxCodec"+path),
		&events3.SinkProps{
			Bucket: stack.Inbox,
			EventSource: &awslambdaeventsources.S3EventSourceProps{
				Events: &[]awss3.EventType{
					awss3.EventType_OBJECT_CREATED,
				},
				Filters: &[]*awss3.NotificationKeyFilter{
					{Prefix: jsii.String(path)},
				},
			},
			Function: &scud.FunctionGoProps{
				SourceCodeModule: "github.com/fogfish/medium",
				SourceCodeLambda: "cmd/lambda/inbox",
				FunctionProps: &awslambda.FunctionProps{
					FunctionName:           jsii.String(name),
					Timeout:                props.Deadline,
					DeadLetterQueueEnabled: jsii.Bool(true),
					DeadLetterQueue:        stack.dlq,
					MemorySize:             props.MemorySize,
					Environment:            &envs,
				},
			},
		},
	)
	stack.Inbox.GrantRead(sink.Handler, nil)
	props.Media.GrantWrite(sink.Handler, nil, nil)
	if props.EventBus != nil {
		props.EventBus.GrantPutEventsTo(sink.Handler, nil)
	}
}
