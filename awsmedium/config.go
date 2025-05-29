//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package awsmedium

import (
	"github.com/fogfish/medium"
)

//
// Configures the stack properties based on the context
//

var (
	//
	// Content centric profile, content is bucketed by type
	ProfilePhoto = medium.Profiles(
		//
		// Avatar
		medium.On("av").
			Process(
				medium.ScaleTo("small", 128, 128),
				medium.ScaleTo("avatar", 400, 400),
				medium.Replica("origin"),
			),
		//
		// Wallpaper
		medium.On("wp").
			Process(
				medium.ScaleTo("equal", 1080, 1080),
				medium.Replica("origin"),
			),
		//
		// Digital Photo
		medium.On("dp").
			Process(
				medium.ScaleTo("small", 128, 128),
				medium.ScaleTo("thumb", 240, 240),
				medium.ScaleTo("cover", 480, 720),
				medium.ScaleTo("equal", 1080, 1080),
				medium.ScaleTo("large", 1080, 1920),
				medium.Replica("origin"),
			),
	)

	//
	// Creator centric profile
	ProfileAuthor = medium.Profiles(
		//
		// Avatar
		medium.Of("av.jpg").
			Process(
				medium.ScaleTo("small", 128, 128),
				medium.ScaleTo("avatar", 400, 400),
				medium.Replica("origin"),
			),
		//
		// Wallpaper
		medium.Of("wp.jpg").
			Process(
				medium.ScaleTo("equal", 1080, 1080),
				medium.Replica("origin"),
			),
		//
		// Digital Photo
		medium.Of("dp.jpg").
			Process(
				medium.ScaleTo("small", 128, 128),
				medium.ScaleTo("thumb", 240, 240),
				medium.ScaleTo("cover", 480, 720),
				medium.ScaleTo("equal", 1080, 1080),
				medium.ScaleTo("large", 1080, 1920),
				medium.Replica("origin"),
			),
	)

	Profiles = map[string][]medium.Profile{
		"photo":  ProfilePhoto,
		"author": ProfileAuthor,
	}
)
