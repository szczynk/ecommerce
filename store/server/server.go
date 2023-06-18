package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"store-go/helper/timeout"
	"syscall"

	"github.com/rs/zerolog"
)

func Run(srv *http.Server, logger *zerolog.Logger) error {
	errChan := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("ListenAndServe(): %w", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-sigs:
		logger.Debug().Msgf("sig: %v", sig)
	}
	logger.Debug().Msg("shutdown server...")

	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	logger.Debug().Msg("server exiting")

	return nil
}
