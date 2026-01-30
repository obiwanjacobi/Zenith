package zsm

// CallGraph represents the function call relationships in the program
type CallGraph struct {
	edges map[string][]string // caller -> list of callees
}

// NewCallGraph creates a new call graph
func NewCallGraph() *CallGraph {
	return &CallGraph{
		edges: make(map[string][]string),
	}
}

// AddFunction registers a function in the call graph
func (cg *CallGraph) AddFunction(name string) {
	if _, exists := cg.edges[name]; !exists {
		cg.edges[name] = []string{}
	}
}

// AddCall records a function call from caller to callee
func (cg *CallGraph) AddCall(caller, callee string) {
	// Ensure caller exists
	if _, exists := cg.edges[caller]; !exists {
		cg.edges[caller] = []string{}
	}

	// Check if this call already exists to avoid duplicates
	for _, existing := range cg.edges[caller] {
		if existing == callee {
			return // Already recorded
		}
	}

	// Add the call
	cg.edges[caller] = append(cg.edges[caller], callee)
}

// GetCallees returns the list of functions called by the given function
func (cg *CallGraph) GetCallees(caller string) []string {
	if callees, exists := cg.edges[caller]; exists {
		return callees
	}
	return []string{}
}

// GetAllFunctions returns all functions in the call graph
func (cg *CallGraph) GetAllFunctions() []string {
	funcs := make([]string, 0, len(cg.edges))
	for name := range cg.edges {
		funcs = append(funcs, name)
	}
	return funcs
}

// GetEdges returns the raw call graph edges (caller -> callees)
func (cg *CallGraph) GetEdges() map[string][]string {
	return cg.edges
}
