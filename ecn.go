package netsim

type ECN uint8

const (
	ECNNonECT ECN = iota
	ECNECT1
	ECNECT0
	ECNCE
)
