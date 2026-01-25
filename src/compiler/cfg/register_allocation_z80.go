package cfg

// Z80Registers defines the available registers for Z80 architecture
var Z80Registers = []Register{
	{Name: "A", Size: 8, Class: RegisterClassAccumulator},
	{Name: "B", Size: 8, Class: RegisterClassGeneral},
	{Name: "C", Size: 8, Class: RegisterClassGeneral},
	{Name: "D", Size: 8, Class: RegisterClassGeneral},
	{Name: "E", Size: 8, Class: RegisterClassGeneral},
	{Name: "H", Size: 8, Class: RegisterClassGeneral},
	{Name: "L", Size: 8, Class: RegisterClassGeneral},
}
