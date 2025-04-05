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
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/go-cdk-easycdn/easycdn"
	"github.com/fogfish/medium"
	"github.com/fogfish/scud"
	"github.com/fogfish/swarm/broker/events3"

	// Note: required to import engine so that all deps used it are lifted to client.
	//       app that uses only stack fails to build if image manipulation library is not imported.
	//       e.g. github.com/anthonynsimon/bild
	_ "github.com/fogfish/medium/internal/awslambda/inbox"
)

type StackProps struct {
	*awscdk.StackProps

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

	// Fully Qualified Domain Name where CDN is hosted (mandatory).
	//
	Site *string

	// ARN of TLS Certificated to enable HTTPS on CDN
	//
	TlsCertificateArn *string

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
}

func (props *StackProps) assert() {
	if len(props.Profiles) == 0 {
		panic("\n\nMedia processing profiles are not defined.")
	}

	if props.Site == nil || *props.Site == "" {
		panic("\n\nFully Qualified Domain Name for CDN is not defined.")
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

type Stack struct {
	awscdk.Stack
	namespace    string
	dlq          awssqs.Queue
	Distribution awscloudfront.Distribution
	Inbox        awss3.Bucket
	Media        awss3.Bucket
}

func NewStack(app awscdk.App, id *string, props *StackProps) *Stack {
	props.assert()

	stack := &Stack{
		Stack:     awscdk.NewStack(app, id, props.StackProps),
		namespace: *id,
	}

	stack.createDLQ(props)

	stack.createInboxBucket(props)
	stack.createMediaBucket(props)

	for _, profile := range props.Profiles {
		stack.createInboxCodec(props, profile)
	}

	stack.createCloudFront(props)

	return stack
}

func (stack *Stack) resource(id string) string {
	return stack.namespace + "-" + id
}

func (stack *Stack) createDLQ(props *StackProps) {
	name := stack.resource("dlq")

	stack.dlq = awssqs.NewQueue(stack.Stack, jsii.String("DeadLetterQueue"),
		&awssqs.QueueProps{
			QueueName:       jsii.String(name),
			RetentionPeriod: props.FailureRetention,
		},
	)
}

func (stack *Stack) createInboxBucket(props *StackProps) {
	name := stack.resource("inbox")

	stack.Inbox = awss3.NewBucket(stack.Stack, jsii.String("Inbox"),
		&awss3.BucketProps{
			BucketName: jsii.String(name),
		},
	)

	stack.Inbox.AddLifecycleRule(&awss3.LifecycleRule{
		Id:         jsii.String("Garbage collector"),
		Enabled:    jsii.Bool(true),
		Expiration: props.Expiration,
	})
}

func (stack *Stack) createMediaBucket(_ *StackProps) {
	name := stack.resource("media")

	stack.Media = awss3.NewBucket(stack.Stack, jsii.String("Media"),
		&awss3.BucketProps{
			BucketName: jsii.String(name),
		},
	)
}

func (stack *Stack) createInboxCodec(props *StackProps, profile medium.Profile) {
	path := profile.Path
	name := stack.resource("inbox-codec-" + filepath.Base(path))
	tout := props.Deadline.ToSeconds(&awscdk.TimeConversionOptions{})

	envs := map[string]*string{
		"CONFIG_STORE_INBOX":          stack.Inbox.BucketName(),
		"CONFIG_STORE_MEDIA":          stack.Media.BucketName(),
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
	stack.Media.GrantWrite(sink.Handler, nil, nil)
	if props.EventBus != nil {
		props.EventBus.GrantPutEventsTo(sink.Handler, nil)
	}
}

func (stack *Stack) createCloudFront(props *StackProps) {
	stack.Distribution = easycdn.NewCdn(stack.Stack, jsii.String("CloudFront"),
		&easycdn.CdnProps{
			Bucket:            stack.Media,
			Site:              props.Site,
			HttpVersion:       awscloudfront.HttpVersion_HTTP2_AND_3,
			TlsCertificateArn: props.TlsCertificateArn,
		},
	).Distribution()
}
