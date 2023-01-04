package ticker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v3/payload"
	"github.com/roadrunner-server/sdk/v3/pool"
	"github.com/roadrunner-server/sdk/v3/pool/static_pool"
	"github.com/roadrunner-server/sdk/v3/worker"
	"go.uber.org/zap"
)

type Configurer interface {
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error

	// Has checks if config section exists.
	Has(name string) bool
}

// Server creates workers for the application.
type Server interface {
	NewPool(ctx context.Context, cfg *pool.Config, env map[string]string, _ *zap.Logger) (*static_pool.Pool, error)
}

type Pool interface {
	// Workers returns worker list associated with the pool.
	Workers() (workers []*worker.Process)

	// Exec payload
	Exec(ctx context.Context, p *payload.Payload) (*payload.Payload, error)

	// Reset kill all workers inside the watcher and replaces with new
	Reset(ctx context.Context) error

	// Destroy all underlying stack (but let them to complete the task).
	Destroy(ctx context.Context)
}

const (
	rrMode     string = "RR_MODE"
	pluginName string = "ticker"
)

type Plugin struct {
	mu     sync.RWMutex
	cfg    *Config
	server Server
	stopCh chan struct{}
	pool   Pool
}

func (p *Plugin) Init(cfg Configurer, server Server) error {
	if !cfg.Has(pluginName) {
		return errors.E(errors.Disabled)
	}

	// have the config
	err := cfg.UnmarshalKey(pluginName, &p.cfg)
	if err != nil {
		return err
	}

	p.cfg.InitDefaults()

	p.stopCh = make(chan struct{}, 1)
	p.server = server

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)

	var err error
	p.mu.Lock()
	p.pool, err = p.server.NewPool(context.Background(), p.cfg.Pool, map[string]string{rrMode: pluginName}, nil)
	p.mu.Unlock()

	if err != nil {
		errCh <- err
		return errCh
	}

	go func() {
		var numTicks = 0
		var lastTick time.Time
		ticker := time.NewTicker(p.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.mu.RLock()
				_, err2 := p.pool.Exec(context.Background(), &payload.Payload{
					Context: []byte(fmt.Sprintf(`{"lastTick": %v}`, lastTick.Unix())),
					Body:    []byte(fmt.Sprintf(`{"tick": %v}`, numTicks)),
				})
				p.mu.RUnlock()
				if err != nil {
					errCh <- err2
					return
				}

				numTicks++
				lastTick = time.Now()
			}
		}
	}()

	return errCh
}

func (p *Plugin) Reset() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.pool == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := p.pool.Reset(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) Stop() error {
	p.stopCh <- struct{}{}
	return nil
}

func (p *Plugin) Name() string {
	return pluginName
}

func (p *Plugin) Weight() uint {
	return 10
}
