package contextaccessor

//go:generate mockgen -package=$GOPACKAGE -destination=../../mocks/$GOPACKAGE/mock_$GOFILE   github.com/fluffy-bunny/fluffycore/echo/contracts/$GOPACKAGE IEchoContextAccessor,IInternalEchoContextAccessor

import (
	"github.com/labstack/echo/v5"
)

type (
	// IEchoContextAccessor ...
	IEchoContextAccessor interface {
		GetContext() *echo.Context
	}
	// IInternalEchoContextAccessor ...
	IInternalEchoContextAccessor interface {
		IEchoContextAccessor
		SetContext(*echo.Context)
	}
)
