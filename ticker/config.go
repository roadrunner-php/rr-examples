package ticker

import (
	"time"

	"github.com/roadrunner-server/sdk/v3/pool"
)

type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	Pool     *pool.Config  `mapstructure:"pool"`
}

func (c *Config) InitDefaults() {
	if c.Pool == nil {
		c.Pool = &pool.Config{}
	}

	c.Pool.InitDefaults()

	// 1s 1h 1m
	if c.Interval == 0 {
		c.Interval = time.Second
	}
}
