//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package main

import (
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/medium/awsmedium"
	"github.com/fogfish/tagver"
)

//
//	cdk deploy \
//	  -c vsn=medium@pr00 \
//	  -c config=photo \
//	  -c site=foobar.example.com \
//	  -c tls-cert-arn=arn:aws:acm:us-east-1:000000000000:certificate/dad...cafe
//

func main() {
	app := awscdk.NewApp(nil)
	vsn := FromContextVsn(app)
	cfg := FromContextCfg(app)
	config := &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
			Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
		},
	}

	edge := awsmedium.NewEdge(app, jsii.String("medium-edge"),
		&awsmedium.EdgeProps{
			StackProps:        config,
			Namespace:         "medium",
			Version:           vsn.Get("medium", "main"),
			Site:              jsii.String(FromContext(app, "site")),
			TlsCertificateArn: jsii.String(FromContext(app, "tls-cert-arn")),
		},
	)

	awsmedium.NewCodec(app, jsii.String("medium-codec"),
		&awsmedium.CodecProps{
			StackProps: config,
			Namespace:  "medium",
			Version:    vsn.Get("medium", "main"),
			Profiles:   awsmedium.Profiles[cfg],
			MemorySize: jsii.Number(512),
			Media:      edge.Media,
		},
	)

	app.Synth(nil)
}

func FromContextVsn(app awscdk.App) tagver.Versions {
	return tagver.NewVersions(FromContext(app, "vsn"))
}

func FromContextCfg(app awscdk.App) string {
	uid := FromContext(app, "config")

	if uid == "" {
		sb := strings.Builder{}
		sb.WriteString("\n\nConfig is not defined, define one of the following configs in the context `-c config=...`\n")
		for profile := range awsmedium.Profiles {
			sb.WriteString("  - " + profile + "\n")
		}
		sb.WriteString("  - [optionally] specify own configuration in cloud/config.go\n")

		panic(sb.String())
	}

	return uid
}

func FromContext(app awscdk.App, key string) string {
	val := app.Node().TryGetContext(jsii.String(key))
	switch v := val.(type) {
	case string:
		return v
	default:
		return ""
	}
}
