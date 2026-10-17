package deploy

import (
	"context"
	"sync"
)

var (
	sshCtxMu sync.RWMutex
	sshCtx   = context.Background()
)

// WithSSHContext runs fn with SSH helpers bound to ctx (CommandContext cancel).
func WithSSHContext(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	sshCtxMu.Lock()
	prev := sshCtx
	sshCtx = ctx
	sshCtxMu.Unlock()
	defer func() {
		sshCtxMu.Lock()
		sshCtx = prev
		sshCtxMu.Unlock()
	}()
	return fn()
}

func currentSSHCtx() context.Context {
	sshCtxMu.RLock()
	defer sshCtxMu.RUnlock()
	if sshCtx == nil {
		return context.Background()
	}
	return sshCtx
}
