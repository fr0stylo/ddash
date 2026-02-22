package servicecatalog

type GraphNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DependencyGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
