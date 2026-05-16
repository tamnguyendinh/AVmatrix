package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
)

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 4848
)

type ListenConfig struct {
	Host string
	Port int
	API  Config
}

func ListenAndServe(ctx context.Context, config ListenConfig) error {
	if config.Host == "" {
		config.Host = DefaultHost
	}
	if config.Port == 0 {
		config.Port = DefaultPort
	}

	server := &http.Server{
		Addr:    net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Handler: NewHandler(config.API),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownErr := server.Shutdown(context.Background())
		listenErr := <-errCh
		if shutdownErr != nil {
			return shutdownErr
		}
		return listenErr
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen on %s failed: %w", server.Addr, err)
		}
		return nil
	}
}
