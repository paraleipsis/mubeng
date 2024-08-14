package server

import (
	"context"
	"os"
	"time"
)

// Stop will terminate proxy server
func Stop(ctx context.Context) {
	_ = server.Shutdown(ctx)
}

func interrupt(sig chan os.Signal) {
	<-sig
	log.Warn("Interuppted. Exiting...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	Stop(ctx)
}
