package contextual

type ContextKeyBool string

func (c *Cancellable) SetContextKey(key ContextKeyBool, value bool) {
	c.AddValue(key, value)
}

func (c *Cancellable) RunIf(key ContextKeyBool, f func()) {
	// Use GetE to access values from c.values sync.Map, consistent with SetContextKey.
	if v, found := c.GetE(key); found {
		if boolVal, isBool := v.(bool); isBool && boolVal {
			f()
		}
	}
}
