package cfg

import "zenith/compiler/zsm"

// FrameSlot represents a variable's fixed location in the function stack frame.
type FrameSlot struct {
	Symbol *zsm.Symbol
	Name   string
	Offset uint16
	Size   uint16
}

// StackFrame manages stack-frame slots and tracks the next free offset.
type StackFrame struct {
	slots      map[*zsm.Symbol]*FrameSlot
	nextOffset uint16
	spillCount int // number of anonymous spill slots added by the register allocator
}

// NewStackFrame creates an empty stack frame starting at offset 0.
func NewStackFrame() *StackFrame {
	return &StackFrame{
		slots:      make(map[*zsm.Symbol]*FrameSlot),
		nextOffset: 0,
	}
}

func (fl *StackFrame) IsEmpty() bool {
	return fl.nextOffset == 0
}

func (fl *StackFrame) Size() uint16 {
	return fl.nextOffset
}

// AddSlot adds a frame slot for a symbol and advances nextOffset.
// If the symbol already has a slot, the existing slot-offset is returned unchanged.
func (fl *StackFrame) AddSlot(symbol *zsm.Symbol, size uint16) uint16 {
	if slot, ok := fl.slots[symbol]; ok {
		return slot.Offset
	}

	slot := &FrameSlot{
		Symbol: symbol,
		Name:   symbol.Name,
		Offset: fl.nextOffset,
		Size:   size,
	}

	fl.slots[symbol] = slot
	fl.nextOffset += size
	return slot.Offset
}

// GetSlot returns the frame slot for a symbol.
func (fl *StackFrame) GetSlot(symbol *zsm.Symbol) (*FrameSlot, bool) {
	slot, ok := fl.slots[symbol]
	return slot, ok
}

// HasSlot returns true if a symbol has an allocated frame slot.
func (fl *StackFrame) HasSlot(symbol *zsm.Symbol) bool {
	_, ok := fl.slots[symbol]
	return ok
}

// AddSpillSlot allocates an anonymous stack slot for a spilled register.
// Returns the byte offset from SP at function entry.
func (fl *StackFrame) AddSpillSlot(size uint16) uint16 {
	offset := fl.nextOffset
	fl.nextOffset += size
	fl.spillCount++
	return offset
}

// HasSpillSlots reports whether the register allocator spilled any VRs into
// the stack frame. Used by SelectPrologue to decide whether to set up IX.
func (fl *StackFrame) HasSpillSlots() bool {
	return fl.spillCount > 0
}
