package netsim

type AttributesKey int

const (
	AttributesKeyECN AttributesKey = iota
)

type Attributes map[any]any
