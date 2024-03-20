package health

type Health interface {
	Start(name string) Item
	Stop(name string)
	Status() map[string]bool
}
