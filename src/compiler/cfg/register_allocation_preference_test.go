package cfg

import (
	"testing"

	"zenith/compiler/zir"
)

func Test_RegisterPreference_ArithmeticPreference(t *testing.T) {
	// 8-bit variable used for arithmetic operations should prefer A register
	usage := zir.VarInitConstant
	usage.AddFlag(zir.VarUsedArithmetic)

	regA := Register{Name: "A", Size: 8, Class: RegisterClassAccumulator}
	regB := Register{Name: "B", Size: 8, Class: RegisterClassGeneral}
	regH := Register{Name: "H", Size: 8, Class: RegisterClassIndex}

	scoreA := calculateRegisterPreference(&regA, usage, 8)
	scoreB := calculateRegisterPreference(&regB, usage, 8)
	scoreH := calculateRegisterPreference(&regH, usage, 8)

	// A should have highest score for arithmetic
	if scoreA <= scoreB || scoreA <= scoreH {
		t.Errorf("A register should have highest score for arithmetic. Scores: A=%d, B=%d, H=%d", scoreA, scoreB, scoreH)
	}

	t.Logf("8-bit arithmetic variable scores: A=%d, B=%d, H=%d", scoreA, scoreB, scoreH)
}

func Test_RegisterPreference_PointerPreference8Bit(t *testing.T) {
	// 8-bit variable used for pointer/indirect addressing
	usage := zir.VarInitPointer
	usage.AddFlag(zir.VarUsedPointer)

	regA := Register{Name: "A", Size: 8, Class: RegisterClassAccumulator}
	regH := Register{Name: "H", Size: 8, Class: RegisterClassIndex}
	regL := Register{Name: "L", Size: 8, Class: RegisterClassIndex}
	regD := Register{Name: "D", Size: 8, Class: RegisterClassGeneral}

	scoreA := calculateRegisterPreference(&regA, usage, 8)
	scoreH := calculateRegisterPreference(&regH, usage, 8)
	scoreL := calculateRegisterPreference(&regL, usage, 8)
	scoreD := calculateRegisterPreference(&regD, usage, 8)

	// H or L should have highest score for pointers
	if scoreH <= scoreA || scoreL <= scoreA {
		t.Errorf("H/L registers should have higher score than A for pointers. Scores: A=%d, H=%d, L=%d, D=%d", scoreA, scoreH, scoreL, scoreD)
	}

	t.Logf("8-bit pointer variable scores: A=%d, H=%d, L=%d, D=%d", scoreA, scoreH, scoreL, scoreD)
}

func Test_RegisterPreference_PointerPreference16Bit(t *testing.T) {
	// 16-bit variable used for pointer should prefer HL pair
	usage := zir.VarInitPointer
	usage.AddFlag(zir.VarUsedPointer)

	regHL := Register{Name: "HL", Size: 16, Class: RegisterClassIndex}
	regDE := Register{Name: "DE", Size: 16, Class: RegisterClassGeneral}
	regBC := Register{Name: "BC", Size: 16, Class: RegisterClassGeneral}

	scoreHL := calculateRegisterPreference(&regHL, usage, 16)
	scoreDE := calculateRegisterPreference(&regDE, usage, 16)
	scoreBC := calculateRegisterPreference(&regBC, usage, 16)

	// HL should have highest score for 16-bit pointers
	if scoreHL <= scoreDE || scoreHL <= scoreBC {
		t.Errorf("HL pair should have highest score for 16-bit pointers. Scores: HL=%d, DE=%d, BC=%d", scoreHL, scoreDE, scoreBC)
	}

	t.Logf("16-bit pointer variable scores: HL=%d, DE=%d, BC=%d", scoreHL, scoreDE, scoreBC)
}

func Test_RegisterPreference_CounterPreference(t *testing.T) {
	// 8-bit variable used as loop counter should prefer B or C
	usage := zir.VarInitCounter
	usage.AddFlag(zir.VarUsedCounter)

	regA := Register{Name: "A", Size: 8, Class: RegisterClassAccumulator}
	regB := Register{Name: "B", Size: 8, Class: RegisterClassGeneral}
	regC := Register{Name: "C", Size: 8, Class: RegisterClassGeneral}
	regD := Register{Name: "D", Size: 8, Class: RegisterClassGeneral}

	scoreA := calculateRegisterPreference(&regA, usage, 8)
	scoreB := calculateRegisterPreference(&regB, usage, 8)
	scoreC := calculateRegisterPreference(&regC, usage, 8)
	scoreD := calculateRegisterPreference(&regD, usage, 8)

	// B or C should have highest score for counters (Z80's DJNZ uses B)
	if scoreB <= scoreA || scoreC <= scoreA {
		t.Errorf("B/C registers should have higher score than A for counters. Scores: A=%d, B=%d, C=%d, D=%d", scoreA, scoreB, scoreC, scoreD)
	}

	t.Logf("8-bit counter variable scores: A=%d, B=%d, C=%d, D=%d", scoreA, scoreB, scoreC, scoreD)
}

func Test_RegisterPreference_SizeMatching(t *testing.T) {
	// Test that register size matching is prioritized
	usage := zir.VarInitConstant // No special usage

	reg8 := Register{Name: "B", Size: 8, Class: RegisterClassGeneral}
	reg16 := Register{Name: "BC", Size: 16, Class: RegisterClassGeneral}

	// 8-bit variable should strongly prefer 8-bit register
	score8for8 := calculateRegisterPreference(&reg8, usage, 8)
	score16for8 := calculateRegisterPreference(&reg16, usage, 8)

	if score8for8 <= score16for8 {
		t.Errorf("8-bit register should have higher score for 8-bit variable. Scores: 8-bit=%d, 16-bit=%d", score8for8, score16for8)
	}

	// 16-bit variable should strongly prefer 16-bit register
	score8for16 := calculateRegisterPreference(&reg8, usage, 16)
	score16for16 := calculateRegisterPreference(&reg16, usage, 16)
	if score16for16 <= score8for16 {
		t.Errorf("16-bit register should have higher score for 16-bit variable. Scores: 8-bit=%d, 16-bit=%d", score8for16, score16for16)
	}

	t.Logf("Size matching: 8-bit var with 8-bit reg=%d, with 16-bit reg=%d", score8for8, score16for8)
	t.Logf("Size matching: 16-bit var with 8-bit reg=%d, with 16-bit reg=%d", score8for16, score16for16)
}

func Test_SelectBestRegister(t *testing.T) {
	// Test selecting best register based on usage for 8-bit arithmetic
	usage := zir.VarInitNone
	usage.AddFlag(zir.VarUsedArithmetic)

	registers := Z80Registers
	usedColors := make(map[int]bool)

	// Select best register for 8-bit arithmetic variable
	bestIdx := selectBestRegister("x", usage, 8, registers, usedColors)
	if bestIdx < 0 {
		t.Fatal("Expected a register to be selected")
	}

	bestReg := registers[bestIdx]
	if bestReg.Name != "A" {
		t.Errorf("Expected A register for 8-bit arithmetic, got %s", bestReg.Name)
	}

	// Now mark A as used and try again
	usedColors[bestIdx] = true
	secondBestIdx := selectBestRegister("y", usage, 8, registers, usedColors)
	if secondBestIdx < 0 {
		t.Fatal("Expected a second-best register to be selected")
	}

	secondBestReg := registers[secondBestIdx]
	t.Logf("Second best for 8-bit arithmetic: %s", secondBestReg.Name)

	// Second best should not be A
	if secondBestReg.Name == "A" {
		t.Error("Second best should not be A since it's already used")
	}
}

func Test_SelectBestRegister16Bit(t *testing.T) {
	// Test selecting best register for 16-bit pointer
	usage := zir.VarInitPointer
	usage.AddFlag(zir.VarUsedPointer)

	registers := Z80Registers
	usedColors := make(map[int]bool)

	// Select best register for 16-bit pointer variable
	bestIdx := selectBestRegister("ptr", usage, 16, registers, usedColors)
	if bestIdx < 0 {
		t.Fatal("Expected a register to be selected")
	}

	bestReg := registers[bestIdx]
	if bestReg.Name != "HL" {
		t.Errorf("Expected HL register for 16-bit pointer, got %s", bestReg.Name)
	}

	t.Logf("Best register for 16-bit pointer: %s (size=%d)", bestReg.Name, bestReg.Size)
}
