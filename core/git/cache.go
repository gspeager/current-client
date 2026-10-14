package git

import "golang.org/x/sync/singleflight"

// flights makes callers that ask the same question at the same time share
// one git process.
var flights singleflight.Group
