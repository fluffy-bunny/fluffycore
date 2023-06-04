package contextaccessor

import (
	"github.com/labstack/echo/v4"
)

type (
	// IEchoContextAccessor ...
	IEchoContextAccessor interface {
		GetContext() echo.Context
	}
	// IInternalEchoContextAccessor ...
	IInternalEchoContextAccessor interface {
		IEchoContextAccessor
		SetContext(echo.Context)
	}
)
