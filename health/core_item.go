package health

type CoreItem struct {
	core *Core
	name string
}

func NewCoreItem(core *Core, name string) *CoreItem {
	return &CoreItem{
		core: core,
		name: name,
	}
}

func (h *CoreItem) Stop() {
	h.core.Stop(h.name)
}
