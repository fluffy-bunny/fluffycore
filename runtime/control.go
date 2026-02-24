package runtime

import (
	"context"
	"fmt"
	"net/http"
	"os"
	go_runtime "runtime"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/reugn/async"
	zlog "github.com/rs/zerolog/log"
)

// Control is the control object that manages an echo web server
type Control struct {
	waitChannel chan os.Signal
	future      async.Future[string]
	e           *echo.Echo
	cancel      context.CancelFunc
	runtime     *Runtime
}

// NewControl creates a new control object
func NewControl(runtime *Runtime) *Control {
	return &Control{
		waitChannel: make(chan os.Signal),
		runtime:     runtime,
	}
}

// Stop ...
func (s *Control) Stop() {
	if s.future == nil {
		return
	}
	zlog.Info().Msg("Stopping Control Web Server")
	if s.cancel != nil {
		s.cancel()
	}
	s.future.Join()
	zlog.Info().Msg("Control Web Server stopped")
}

// Start starts the echo web server using async and futures
func (s *Control) Start() {
	controlPort := os.Getenv("CONTROL_PORT")
	if len(controlPort) != 0 {
		// convert to int
		port, err := strconv.Atoi(controlPort)
		if err != nil {
			zlog.Fatal().Err(err).Msg("Failed to convert Control port to int")
		}
		// start the control server
		zlog.Info().Int("port", port).Msg("Starting Control server")

		s.e = echo.New()
		e := s.e
		e.GET("/", func(c *echo.Context) error {
			return c.String(http.StatusOK, "Hello from Control")
		})
		e.GET("/stop", func(c *echo.Context) error {
			s.runtime.Stop()
			return c.String(http.StatusOK, "Signalled server to stop")
		})
		e.GET("/gc", func(c *echo.Context) error {
			go_runtime.GC()
			return c.String(http.StatusOK, "Called GC")
		})
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		asyncAction := func() async.Future[string] {
			promise := async.NewPromise[string]()
			go func() {
				addr := fmt.Sprintf(":%d", port)
				sc := echo.StartConfig{Address: addr}
				if err := sc.Start(ctx, e); err != nil {
					zlog.Error().Err(err).Msg("control server error")
					promise.Failure(err)
				} else {
					promise.Success("OK")
				}
			}()

			return promise.Future()
		}
		s.future = asyncAction()
	}
}
