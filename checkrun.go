package contextual

type ContextKeyBool string

func (c *Cancellable) SetContextKey(key ContextKeyBool, value bool) {
	c.AddValue(key, value)
}

func (c *Cancellable) RunIf(key ContextKeyBool, f func()) {
	if v, ok := c.Value(key).(bool); ok && v {
		f()
	}
}
