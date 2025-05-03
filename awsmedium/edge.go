package awsmedium

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/go-cdk-easycdn/easycdn"
	"github.com/fogfish/tagver"
)

type EdgeProps struct {
	*awscdk.StackProps

	// Namespace for cloud resource provisioning
	Namespace string

	// Version of the deployment
	Version tagver.Version

	// Fully Qualified Domain Name where CDN is hosted (mandatory).
	//
	Site *string

	// ARN of TLS Certificated to enable HTTPS on CDN
	//
	TlsCertificateArn *string
}

// The distribution edge of media content
type Edge struct {
	awscdk.Stack
	namespace    string
	version      tagver.Version
	Distribution awscloudfront.Distribution
	Media        awss3.Bucket
}

// The distribution edge of media content
func NewEdge(app awscdk.App, id *string, props *EdgeProps) *Edge {
	if props.Site == nil || *props.Site == "" {
		panic("\n\nFully Qualified Domain Name for CDN is not defined.")
	}

	stack := &Edge{
		Stack:     awscdk.NewStack(app, id, props.StackProps),
		namespace: props.Namespace,
		version:   props.Version,
	}

	stack.createMediaBucket(props)
	stack.createCloudFront(props)

	return stack
}

func (edge *Edge) resource(id string) string {
	return edge.version.Tag(edge.namespace + "-" + id)
}

func (edge *Edge) createMediaBucket(_ *EdgeProps) {
	name := edge.resource("media")

	edge.Media = awss3.NewBucket(edge.Stack, jsii.String("Media"),
		&awss3.BucketProps{
			BucketName: jsii.String(name),
		},
	)
}

func (edge *Edge) createCloudFront(props *EdgeProps) {
	edge.Distribution = easycdn.NewCdn(edge.Stack, jsii.String("CloudFront"),
		&easycdn.CdnProps{
			Bucket:            edge.Media,
			Site:              props.Site,
			HttpVersion:       awscloudfront.HttpVersion_HTTP2_AND_3,
			TlsCertificateArn: props.TlsCertificateArn,
		},
	).Distribution()
}
