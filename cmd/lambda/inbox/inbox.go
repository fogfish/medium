//
// Copyright (C) 2023 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/medium
//

package main

import (
	"github.com/fogfish/medium/internal/awslambda/inbox"
)

func main() {
	inbox.Runner()
}
