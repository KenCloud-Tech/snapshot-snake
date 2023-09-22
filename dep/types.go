package dep

import (
	"context"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

// GlobalContext is a type alias for standard context.Context that used in dep-injection
type GlobalContext context.Context

type RepoPath string

type SegmentMetaDS dtypes.MetadataDS
