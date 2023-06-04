package time

import (
	"time"

	di "github.com/dozm/di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

type (
	serviceTimeUtils struct {
		Time fluffycore_contracts_common.ITime `inject:""`
	}
)

// AddSingletonITimeUtils ...
func AddSingletonITimeUtils(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.ITimeUtils](builder,
		func(timeService fluffycore_contracts_common.ITime) fluffycore_contracts_common.ITimeUtils {
			return &serviceTimeUtils{
				Time: timeService,
			}
		})
}

// NewTimeUtils ...
func NewTimeUtils(timeService fluffycore_contracts_common.ITime) fluffycore_contracts_common.ITimeUtils {
	return &serviceTimeUtils{
		Time: timeService,
	}
}

// StartOfMonthUTC returns the start of current month in UTC
func (s *serviceTimeUtils) StartOfMonthUTC(offsetMonth int) time.Time {
	now := s.Time.Now()
	currentYear := now.Year()
	nextYear := currentYear
	currentMonth := now.Month()
	tt := time.Date(nextYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	tt = tt.AddDate(0, offsetMonth, 0)
	return tt
}
