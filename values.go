package contextual

import (
	"fmt"
	"strconv"
)

type ContextKV struct {
	Key   any
	Value any
}

func (c *Cancellable) AddValue(key any, value any) {
	c.values.Store(key, value)
}

func (c *Cancellable) GetE(key any) (any, bool) {
	v, ok := c.values.Load(key)
	return v, ok
}

func (c *Cancellable) Get(key any) any {
	if v, ok := c.GetE(key); ok {
		return v
	}

	return nil
}

func (c *Cancellable) GetString(key any) string {
	if v, vok := c.GetE(key); vok {
		if s, sok := v.(string); sok {
			return s
		}

		return fmt.Sprintf("%s", v)
	}

	return ""
}

func (c *Cancellable) GetInt(key any) int {
	if v, vok := c.GetE(key); vok {
		switch i := v.(type) {
		case int:
			return i
		case int16:
			return int(i)
		case int32:
			return int(i)
		case int64:
			return int(i)
		case string:
			o, err := strconv.ParseInt(i, 0, 0)
			if err == nil {
				return int(o)
			}

			return 0
		}
	}

	return 0
}
