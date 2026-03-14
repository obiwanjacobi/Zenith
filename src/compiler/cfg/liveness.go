package cfg

import "sort"

// ============================================================================
// Live range computation (target-independent)
// ============================================================================

// LiveRange records the live interval of a single TempVR in the linearised
// instruction stream of a function. Start and End are inclusive instruction
// indices in DFS pre-order across all blocks (exit block last).
//
// Because VRs are SSA-style (written exactly once), the live range is simply
// [first-definition, last-use]. The allocator uses End to decide which active
// interval to spill when registers are exhausted.
type LiveRange struct {
	VR    *TempVR
	Start int // index of first def or first use (whichever comes first)
	End   int // index of last use
}

// ComputeLiveRanges returns live ranges for every TempVR that appears in the
// function's machine instructions, sorted by Start ascending.
func ComputeLiveRanges(fnCFG *CFG) []LiveRange {
	blocks := dfsOrder(fnCFG)

	starts := make(map[int]int) // TempVR.ID → first position
	ends := make(map[int]int)   // TempVR.ID → last position
	vrByID := make(map[int]*TempVR)

	record := func(op VROperand, pos int) {
		tvr, ok := op.(*TempVR)
		if !ok {
			return
		}
		if _, seen := starts[tvr.ID()]; !seen {
			starts[tvr.ID()] = pos
		}
		ends[tvr.ID()] = pos
		vrByID[tvr.ID()] = tvr
	}

	pos := 0
	for _, block := range blocks {
		for _, mi := range block.MachineInstructions {
			// Record the definition first (so start = def position if it's the first).
			if res := mi.GetResult(); res != nil {
				record(res, pos)
			}
			for _, op := range mi.GetOperands() {
				record(op, pos)
			}
			pos++
		}
	}

	ranges := make([]LiveRange, 0, len(starts))
	for id, start := range starts {
		ranges = append(ranges, LiveRange{
			VR:    vrByID[id],
			Start: start,
			End:   ends[id],
		})
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})
	return ranges
}

// dfsOrder returns blocks in DFS pre-order from the entry block, with the
// exit block appended last. Back edges (e.g. for.inc → for.cond) are skipped
// because the target block was already visited.
func dfsOrder(fnCFG *CFG) []*BasicBlock {
	visited := make(map[int]bool)
	order := make([]*BasicBlock, 0, len(fnCFG.Blocks))

	var visit func(*BasicBlock)
	visit = func(b *BasicBlock) {
		if visited[b.ID] {
			return
		}
		visited[b.ID] = true
		if b == fnCFG.Exit {
			return // deferred to end
		}
		order = append(order, b)
		for _, succ := range b.Successors {
			visit(succ)
		}
	}

	visit(fnCFG.Entry)
	order = append(order, fnCFG.Exit)
	return order
}
