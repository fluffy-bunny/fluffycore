package common

import "context"

type (
	AppContext func() context.Context
)
