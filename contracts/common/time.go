package common

//go:generate mockgen -package=$GOPACKAGE -destination=../../mocks/$GOPACKAGE/mock_$GOFILE   github.com/fluffy-bunny/fluffycore/contracts/common ITimeUtils,ITime

import "time"

type (

	// ITimeUtils ...
	ITimeUtils interface {
		// StartOfMonthUTC where offsetMonth is 0-based (0 = Current Month)
		StartOfMonthUTC(offsetMonth int) time.Time
		// format is "2006-01-02T15:04:05Z07:00"
		//Format(layout string, t time.Time) string
	}

	// ITime ...
	ITime interface {
		Now() time.Time
	}
	// TimeNow ...
	TimeNow func() time.Time
	// TimeParse ...
	TimeParse func(value string) (time.Time, error)
)
