package time

import (
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	mocks_timeutils "github.com/fluffy-bunny/fluffycore/mocks/common"
	gomock "github.com/golang/mock/gomock"
	parsetime "github.com/tkuchiki/parsetime"
)

// AddTimeNow adds a time now that always returns time.Now
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

type serviceTime struct {
}

func newTime() fluffycore_contracts_common.ITime {
	return &serviceTime{}
}

// AddSingletonITime ...
func AddSingletonITime(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.ITime](builder,
		func() fluffycore_contracts_common.ITime {
			return &serviceTime{}
		})
}
func (s *serviceTime) Now() time.Time {
	return time.Now()
}

// AddTimeParse adds a singleton of Parse to the container
func AddTimeParse(builder di.ContainerBuilder) {
	impl := func(value string) (time.Time, error) {
		p, err := parsetime.NewParseTime()
		if err != nil {
			return time.Time{}, err
		}
		return p.Parse(value)
	}
	di.AddSingleton[fluffycore_contracts_common.TimeParse](builder, func() fluffycore_contracts_common.TimeParse {
		return impl
	})

}

var (
	// Months ...
	Months = []time.Month{
		time.January,
		time.February,
		time.March,
		time.April,
		time.May,
		time.June,
		time.July,
		time.August,
		time.September,
		time.October,
		time.November,
		time.December,
	}
)

// NewMockITimeYearMonthDate ...
func NewMockITimeYearMonthDate(ctrl *gomock.Controller, year int, month time.Month) fluffycore_contracts_common.ITime {
	return NewMockITimeDate(ctrl, year, month, 1, 0, 0, 0, 0, time.UTC)
}

// NewMockITimeYearMonthDayDate ...
func NewMockITimeYearMonthDayDate(ctrl *gomock.Controller, year int, month time.Month, day int) fluffycore_contracts_common.ITime {
	return NewMockITimeDate(ctrl, year, month, day, 0, 0, 0, 0, time.UTC)
}

// NewMockITimeYearMonthDayHourDate ...
func NewMockITimeYearMonthDayHourDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int) fluffycore_contracts_common.ITime {
	return NewMockITimeDate(ctrl, year, month, day, hour, 0, 0, 0, time.UTC)
}

// NewMockITimeYearMonthDayHourMinDate ...
func NewMockITimeYearMonthDayHourMinDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int, min int) fluffycore_contracts_common.ITime {
	return NewMockITimeDate(ctrl, year, month, day, hour, min, 0, 0, time.UTC)
}

// NewMockITimeDate ...
func NewMockITimeDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int, min int, sec int, nsec int, loc *time.Location) fluffycore_contracts_common.ITime {
	mockITime := mocks_timeutils.NewMockITime(ctrl)
	mockTimeNow := time.Date(year, month, day, hour, min, sec, nsec, loc)
	mockITime.EXPECT().Now().Return(mockTimeNow).AnyTimes()
	return mockITime
}

// NewMockTimeNowDate ...
func NewMockTimeNowDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int, min int, sec int, nsec int, loc *time.Location) fluffycore_contracts_common.TimeNow {
	mockITime := mocks_timeutils.NewMockITime(ctrl)
	mockTimeNow := time.Date(year, month, day, hour, min, sec, nsec, loc)
	mockITime.EXPECT().Now().Return(mockTimeNow).AnyTimes()
	return func() time.Time {
		return mockITime.Now()
	}
}

// NewMockTimeNowYearMonthDate ...
func NewMockTimeNowYearMonthDate(ctrl *gomock.Controller, year int, month time.Month) fluffycore_contracts_common.TimeNow {
	return NewMockTimeNowDate(ctrl, year, month, 1, 0, 0, 0, 0, time.UTC)
}

// NewMockTimeNowYearMonthDayDate ...
func NewMockTimeNowYearMonthDayDate(ctrl *gomock.Controller, year int, month time.Month, day int) fluffycore_contracts_common.TimeNow {
	return NewMockTimeNowDate(ctrl, year, month, day, 0, 0, 0, 0, time.UTC)
}

// NewMockTimeNowYearMonthDayHourDate ...
func NewMockTimeNowYearMonthDayHourDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int) fluffycore_contracts_common.TimeNow {
	return NewMockTimeNowDate(ctrl, year, month, day, hour, 0, 0, 0, time.UTC)
}

// NewMockTimeNowYearMonthDayHourMinDate ...
func NewMockTimeNowYearMonthDayHourMinDate(ctrl *gomock.Controller, year int, month time.Month, day int, hour int, min int) fluffycore_contracts_common.TimeNow {
	return NewMockTimeNowDate(ctrl, year, month, day, hour, min, 0, 0, time.UTC)
}
