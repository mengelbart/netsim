package bandwdith

import "github.com/mengelbart/netsim"

type Rate int

type Writer struct {
	rate Rate
}

func NewWriter() netsim.Node {
	return nil
}
