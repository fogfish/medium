//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package medium_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/fogfish/medium"
)

func TestResolution(t *testing.T) {
	t.Run("WellFormat", func(t *testing.T) {
		for input, expect := range map[string]medium.Resolution{
			"pixel-1x1":       {Label: "pixel", Width: 1, Height: 1},
			"small-128x128":   {Label: "small", Width: 128, Height: 128},
			"large-1080x1920": {Label: "large", Width: 1080, Height: 1920},
			"origin":          {Label: "origin", Width: 0, Height: 0},
			"o":               {Label: "o", Width: 0, Height: 0},
		} {
			val, err := medium.NewResolution(input)
			it.Then(t).Should(
				it.Nil(err),
				it.Equiv(val, expect),
			)
		}
	})

	t.Run("Corrupted", func(t *testing.T) {
		for _, input := range []string{
			"",
			"small-",
			"small-128",
			"small-128x",
			"small-x128",
			"small-Ax128",
			"small-128xA",
		} {
			_, err := medium.NewResolution(input)
			it.Then(t).ShouldNot(
				it.Nil(err),
			)
		}

	})
}
