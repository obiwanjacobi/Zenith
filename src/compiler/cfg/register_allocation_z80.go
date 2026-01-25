package cfg

// Z80Registers defines the available registers for Z80 architecture
// Includes both single 8-bit registers and 16-bit register pairs
var Z80Registers = []Register{
	// 8-bit single registers
	{Name: "A", Size: 8, Class: RegisterClassAccumulator},
	{Name: "B", Size: 8, Class: RegisterClassGeneral},
	{Name: "C", Size: 8, Class: RegisterClassGeneral},
	{Name: "D", Size: 8, Class: RegisterClassGeneral},
	{Name: "E", Size: 8, Class: RegisterClassGeneral},
	{Name: "H", Size: 8, Class: RegisterClassGeneral},
	{Name: "L", Size: 8, Class: RegisterClassGeneral},

	// 16-bit register pairs
	{Name: "BC", Size: 16, Class: RegisterClassGeneral},
	{Name: "DE", Size: 16, Class: RegisterClassGeneral},
	{Name: "HL", Size: 16, Class: RegisterClassIndex},
}

// isZ80RegisterPair returns true if the register is a Z80 register pair (BC, DE, HL)
// This is Z80-specific knowledge - on Z80, these 16-bit registers are composed of
// two 8-bit registers that can be used independently
func isZ80RegisterPair(reg Register) bool {
	return reg.Size == 16 && (reg.Name == "BC" || reg.Name == "DE" || reg.Name == "HL")
}
