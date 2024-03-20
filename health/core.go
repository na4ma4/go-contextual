package health

import (
	"sync"

	"go.uber.org/zap"
)

type Core struct {
	logger        *zap.Logger
	processActive map[string]bool
	lock          sync.RWMutex
}

func NewCore(logger *zap.Logger) *Core {
	return &Core{
		logger:        logger,
		processActive: map[string]bool{},
	}
}

func (c *Core) Start(name string) Item {
	c.logger.Debug("Start",
		zap.String("farnsworth.health.name", name), zap.Reflect("farnsworth.debug.process", c.processActive),
	)

	c.lock.Lock()
	defer c.lock.Unlock()

	c.processActive[name] = true

	return NewCoreItem(c, name)
}

func (c *Core) Stop(name string) {
	c.logger.Debug("Stop",
		zap.String("farnsworth.health.name", name), zap.Reflect("farnsworth.debug.process", c.processActive),
	)

	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.processActive[name]; ok {
		c.processActive[name] = false
	}
}

func (c *Core) Status() map[string]bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	out := map[string]bool{}

	for k, v := range c.processActive {
		out[k] = v
	}

	return out
}
