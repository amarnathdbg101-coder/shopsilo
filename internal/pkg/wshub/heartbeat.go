package gowshub

import (
	"context"
	"time"
)

// StartHeartbeat initiates a background ping loop on the safe connection.
// It sends a Ping control frame at every pingInterval tick until ctx is canceled or the returned stop func is invoked.
func StartHeartbeat(ctx context.Context, conn *SafeConn, pingInterval, writeTimeout time.Duration) func() {
	if pingInterval <= 0 {
		pingInterval = 25 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 5 * time.Second
	}

	stopCh := make(chan struct{})
	ticker := time.NewTicker(pingInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				deadline := time.Now().Add(writeTimeout)
				if err := conn.WriteControl(PingMessage, nil, deadline); err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	return func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
}
