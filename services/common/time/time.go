package time

import (
	"time"

	"github.com/dozm/di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

// AddTimeNowZero adds a time now that returns 1970-01-01 00:00:00 +0000 UTC
func AddTimeNow(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.TimeNow](b, func() fluffycore_contracts_common.TimeNow {
		return time.Now
	})
}

// AddTimeNow1970 adds a time now that always returns 1970-01-01 00:00:00 +0000 UTC
func AddTimeNow1970(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.TimeNow](b, func() fluffycore_contracts_common.TimeNow {
		return func() time.Time {
			// return 1970-01-01 00:00:00 +0000 UTC
			return time.Unix(0, 0)
		}
	})
}
