//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package awsmedium

import (
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/go-cdk-easycdn/easycdn"
	"github.com/fogfish/medium"
	"github.com/fogfish/scud"
	"github.com/fogfish/swarm/broker/events3"
)

type StackProps struct {
	*awscdk.StackProps

	// Version of the deployment, allowing Green/Blue deployment patterns
	// Default: latest
	//
	Version string

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
}

func (props *StackProps) assert() {
	if props.Profiles == nil || len(props.Profiles) == 0 {
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
	namespace    *string
	dlq          awssqs.Queue
	Distribution awscloudfront.CloudFrontWebDistribution
	Inbox        awss3.Bucket
	Media        awss3.Bucket
}

func NewStack(app awscdk.App, id *string, props *StackProps) *Stack {
	props.assert()

	stackID := *id
	if props.Version != "" {
		stackID += "-" + props.Version
	}

	stack := &Stack{
		Stack:     awscdk.NewStack(app, jsii.String(stackID), props.StackProps),
		namespace: id,
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

func (stack *Stack) resource(props *StackProps, id string) string {
	if props.Version == "" {
		return *stack.namespace + "-" + id
	}

	return *stack.namespace + "-" + id + "-" + props.Version
}

func (stack *Stack) createDLQ(props *StackProps) {
	name := stack.resource(props, "dlq")

	stack.dlq = awssqs.NewQueue(stack.Stack, jsii.String("DeadLetterQueue"),
		&awssqs.QueueProps{
			QueueName:       jsii.String(name),
			RetentionPeriod: props.FailureRetention,
		},
	)
}

func (stack *Stack) createInboxBucket(props *StackProps) {
	name := stack.resource(props, "inbox")

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

func (stack *Stack) createMediaBucket(props *StackProps) {
	name := stack.resource(props, "media")

	stack.Media = awss3.NewBucket(stack.Stack, jsii.String("Media"),
		&awss3.BucketProps{
			BucketName: jsii.String(name),
		},
	)
}

func (stack *Stack) createInboxCodec(props *StackProps, profile medium.Profile) {
	path := profile.Path
	name := stack.resource(props, "inbox-codec-"+path)
	tout := props.Deadline.ToSeconds(&awscdk.TimeConversionOptions{})

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
			Lambda: &scud.FunctionGoProps{
				SourceCodePackage: "github.com/fogfish/medium",
				SourceCodeLambda:  "cmd/lambda/inbox",
				FunctionProps: &awslambda.FunctionProps{
					FunctionName:           jsii.String(name),
					Timeout:                props.Deadline,
					DeadLetterQueueEnabled: jsii.Bool(true),
					DeadLetterQueue:        stack.dlq,
					MemorySize:             props.MemorySize,
					Environment: &map[string]*string{
						"CONFIG_VSN":                  &props.Version,
						"CONFIG_STORE_INBOX":          stack.Inbox.BucketName(),
						"CONFIG_STORE_MEDIA":          stack.Media.BucketName(),
						"CONFIG_CODEC_PROFILE":        jsii.String(profile.String()),
						"CONFIG_SWARM_TIME_TO_FLIGHT": jsii.String(strconv.Itoa(int(*tout))),
					},
				},
			},
		},
	)
	stack.Inbox.GrantRead(sink.Handler, nil)
	stack.Media.GrantWrite(sink.Handler, nil, nil)

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
