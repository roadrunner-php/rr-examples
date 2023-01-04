package ticker

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"

	"github.com/roadrunner-server/config/v3"
	"github.com/roadrunner-server/endure/pkg/container"
	"github.com/roadrunner-server/logger/v3"
	"github.com/roadrunner-server/resetter/v3"
	"github.com/roadrunner-server/server/v3"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_Init(t *testing.T) {
	c, err := endure.NewContainer(nil, endure.SetLogLevel(endure.InfoLevel))
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&Plugin{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&logger.Plugin{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&server.Plugin{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&config.Plugin{
		Prefix: "rr",
		Path:   ".rr.yaml",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&resetter.Plugin{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := c.Serve()
	if err != nil {
		t.Fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = c.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = c.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = c.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	}()

	wg.Wait()
}
