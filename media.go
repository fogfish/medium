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
	"strings"

	"github.com/fogfish/curie"
)

type Media struct {
	HashID      curie.IRI `metadata:"hashkey"`
	SortID      curie.IRI `metadata:"sortkey"`
	ContentType string    `metadata:"Content-Type"`
}

func (file Media) HashKey() curie.IRI { return file.HashID }
func (file Media) SortKey() curie.IRI { return file.SortID }
func (file Media) PathKey() string    { return string(file.HashID + "/" + file.SortID) }

// Parses path into reference to media object
func NewMediaFromPath(path string) (*Media, error) {
	seq := strings.SplitN(path, "/", 2)
	if len(seq) != 2 {
		return nil, fmt.Errorf("path format is not supported: %s", path)
	}

	return &Media{
		HashID: curie.IRI(seq[0]),
		SortID: curie.IRI(seq[1]),
	}, nil
}
