package netsim

type Path struct {
	nodes []Node
}

func NewPath(nodes []Node) *Path {
	return &Path{
		nodes: nodes,
	}
}

func (p *Path) Connect(writer PacketWriter) PacketWriter {
	for _, node := range p.nodes {
		writer = node.Link(writer)
	}
	return writer
}
