//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package medium

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Media encoding profile, ensemble of resolutions builds the profile
// (e.g. avatar profile defines small, medium and large encoding of user's avatar)
type Profile struct {
	Path        string
	Resolutions []Resolution
}

// Profiles is part of config DSL
func Profiles(seq ...Profile) []Profile { return seq }

// Parses resolution from string {Path}:{Name}-{Width}x{Height}:{Name}-{Width}x{Height}
func NewProfile(spec string) (Profile, error) {
	seq := strings.Split(spec, ":")

	resolutions := make([]Resolution, len(seq)-1)

	for i, x := range seq[1:] {
		r, err := NewResolution(x)
		if err != nil {
			return Profile{}, err
		}
		resolutions[i] = r
	}

	return Profile{
		Path:        seq[0],
		Resolutions: resolutions,
	}, nil
}

func (p Profile) String() string {
	seq := make([]string, len(p.Resolutions))
	for i, r := range p.Resolutions {
		seq[i] = r.String()
	}

	return p.Path + ":" + strings.Join(seq, ":")
}

// Media file resolution.
type Resolution struct {
	Label  string
	Width  int
	Height int
}

// Parses resolution from string {Name}-{Width}x{Height}
func NewResolution(spec string) (Resolution, error) {
	if len(spec) == 0 {
		return Resolution{}, fmt.Errorf("invalid resolution: %s", spec)
	}

	seq := strings.Split(spec, "-")
	if len(seq) == 1 {
		return Resolution{Label: spec}, nil
	}

	if len(seq) != 2 {
		return Resolution{}, fmt.Errorf("invalid resolution: %s", spec)
	}

	res := strings.Split(seq[1], "x")
	if len(res) != 2 {
		return Resolution{}, fmt.Errorf("invalid resolution: %s", spec)
	}

	width, err := strconv.Atoi(res[0])
	if err != nil {
		return Resolution{}, fmt.Errorf("invalid resolution: %s", spec)
	}

	height, err := strconv.Atoi(res[1])
	if err != nil {
		return Resolution{}, fmt.Errorf("invalid resolution: %s", spec)
	}

	return Resolution{
		Label:  seq[0],
		Width:  width,
		Height: height,
	}, nil
}

func (r Resolution) String() string {
	if r.Width == 0 && r.Height == 0 {
		return r.Label
	}

	return fmt.Sprintf("%s-%dx%d", r.Label, r.Width, r.Height)
}

func (r Resolution) FileSuffix(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + "." + r.String()
}

//
// Config DSL
//

// `On` defines a key prefix at S3 bucket.
// It triggers processing pipeline when object is uploaded into inbox.
func On(path string) Profile {
	return Profile{Path: path, Resolutions: []Resolution{}}
}

// `Process` defines operation to be executed for media file.
func (p Profile) Process(seq ...Resolution) Profile {
	return Profile{
		Path:        p.Path,
		Resolutions: seq,
	}
}

// ScaleTo processing step scales media into specified resolution
func ScaleTo(label string, w int, h int) Resolution {
	return Resolution{Label: label, Width: w, Height: h}
}

// Replica processing step copies media "almost" as-is
func Replica(label string) Resolution {
	return Resolution{Label: label, Width: 0, Height: 0}
}
