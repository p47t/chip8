package chip8

import (
	"fmt"
	"math/rand"
	"io"
)

const (
	StartAddress = 0x200
	RegCarry     = 0xF
)

type CPU struct {
	V     [16]uint8 // general-purpose registers
	I     uint16    // Index register
	PC    uint16    // program counter
	SP    uint16    // stack pointer
	Stack [16]uint16

	cycles int64
}

func (cpu *CPU) Print(w io.Writer) {
	fmt.Fprintf(w, "Cycles %d\n\n", cpu.cycles)
	fmt.Fprintf(w, "PC = 0x%x\n", cpu.PC)
	fmt.Fprintf(w, "SP = 0x%x\n", cpu.SP)
	fmt.Fprintf(w, " I = 0x%x\n", cpu.I)
	for i := 0; i < len(cpu.V); i++ {
		fmt.Fprintf(w, "V%X = 0x%x\n", i, cpu.V[i])
	}
}

func (cpu *CPU) reset() {
	cpu.PC = StartAddress
	cpu.I = 0
	cpu.SP = 0
	cpu.cycles = 0

	// clear stack
	for i := 0; i < len(cpu.Stack); i++ {
		cpu.Stack[i] = 0
	}

	// clear register V0-VF
	for i := 0; i < len(cpu.V); i++ {
		cpu.V[i] = 0
	}
}

func (cpu *CPU) Cycle(mem *Memory, gfx *Graphics, sys *System) {
	cpu.step(mem, gfx, sys)
	cpu.cycles++
}

// decode and step opcode
func (cpu *CPU) step(mem *Memory, gfx *Graphics, sys *System) {
	opc := mem.fetchOpcode(cpu.PC)
	newPC := cpu.PC + 2

	switch opc & 0xF000 {
	case 0x0000:
		switch opc {
		case 0x00E0: // 0x00E0: Clears the screen
			cpu.clearScreen(gfx)
		case 0x00EE: // 0x00EE: Returns from subroutine
			newPC = cpu.ret()
		default:
			addr := opc & 0xFFF
			cpu.callRCA1802(addr)
		}

	case 0x1000: // 0x1NNN: Jumps to address NNN
		newPC = cpu.jump(opc & 0xFFF)

	case 0x2000: // 0x2NNN: Calls subroutine at NNN.
		newPC = cpu.call(opc & 0xFFF)

	case 0x3000: // 0x3XNN: Skips the next instruction if VX equals NN
		x := uint8((opc & 0x0F00) >> 8)
		val := uint8(opc & 0xFF)
		newPC = cpu.skipEq(x, val)

	case 0x4000: // 0x4XNN: Skips the next instruction if VX doesn't equal NN
		x := uint8((opc & 0x0F00) >> 8)
		val := uint8(opc & 0xFF)
		newPC = cpu.skipNeq(x, val)

	case 0x5000: // 0x5XY0: Skips the next instruction if VX equals VY.
		x := uint8((opc & 0x0F00) >> 8)
		y := uint8((opc & 0x00F0) >> 4)
		newPC = cpu.SkipEqReg(x, y)

	case 0x6000: // 0x6XNN: Sets VX to NN.
		reg := uint8((opc & 0x0F00) >> 8)
		val := uint8(opc & 0x00FF)
		cpu.setV(reg, val)

	case 0x7000: // 0x7XNN: Adds NN to VX.
		reg := uint8((opc & 0x0F00) >> 8)
		val := uint8(opc & 0x00FF)
		cpu.addV(reg, val)

	case 0x8000:
		x := uint8((opc & 0x0F00) >> 8)
		y := uint8((opc & 0x00F0) >> 4)
		switch opc & 0x000F {
		case 0x0000: // 0x8XY0: Sets VX to the value of VY
			cpu.setXY(x, y)
		case 0x0001: // 0x8XY1: Sets VX to "VX OR VY"
			cpu.setXOrY(x, y)
		case 0x0002: // 0x8XY2: Sets VX to "VX AND VY"
			cpu.setXAndY(x, y)
		case 0x0003: // 0x8XY3: Sets VX to "VX XOR VY"
			cpu.setXXorY(x, y)
		case 0x0004: // 0x8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't
			cpu.addXY(x, y)
		case 0x0005: // 0x8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			cpu.subXY(x, y)
		case 0x0006: // 0x8XY6: Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift
			cpu.shiftRightX(x)
		case 0x0007: // 0x8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			cpu.minusXY(x, y)
		case 0x000E: // 0x8XYE: Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift
			cpu.shiftLeft(x)
		default:
			cpu.unknownOp(opc)
		}

	case 0x9000: // 0x9XY0: Skips the next instruction if VX doesn't equal VY
		x := uint8((opc & 0x0F00) >> 8)
		y := uint8((opc & 0x00F0) >> 4)
		newPC = cpu.skipNeqReg(x, y)

	case 0xA000: // ANNN: Sets I to the address NNN
		cpu.mvi(opc & 0xFFF)

	case 0xB000: // BNNN: Jumps to the address NNN plus V0
		newPC = cpu.jv0(opc & 0xFFF)

	case 0xC000: // CXNN: Sets VX to a random number and NN
		reg := uint8((opc & 0x0F00) >> 8)
		val := uint8(opc & 0x00FF)
		cpu.rand(reg, val)

	case 0xD000: // DXYN: Draws a sprite at coordinate (VX, VY)that has a width of 8 pixels and a height of N pixels.
		x := uint8((opc & 0x0F00) >> 8)
		y := uint8((opc & 0x00F0) >> 4)
		n := uint8(opc & 0x000F)
		cpu.drawSprite(mem, gfx, x, y, n)

	case 0xE000:
		x := uint8((opc & 0x0F00) >> 8)
		switch opc & 0x00FF {
		case 0x009E: // EX9E: Skips the next instruction if the key stored in VX is pressed
			newPC = cpu.skipIfKeyPressed(sys, x)
		case 0x00A1: // EXA1: Skips the next instruction if the key stored in VX isn't pressed
			newPC = cpu.skipIfKeyNotPressed(sys, x)
		default:
			cpu.unknownOp(opc)
		}

	case 0xF000:
		x := uint8((opc & 0x0F00) >> 8)
		switch opc & 0x00FF {
		case 0x0007: // FX07: Sets VX to the value of the delay timer
			cpu.getDelayTimer(sys, x)

		case 0x000A: // FX0A: A key press is awaited, and then stored in VX
			newPC = cpu.awaitKey(sys, x)

		case 0x0015: // FX15: Sets the delay timer to VX
			cpu.setDelayTimer(sys, x)

		case 0x0018: // FX18: Sets the sound timer to VX
			cpu.setSoundTimer(sys, x)

		case 0x001E: // FX1E: Adds VX to I
			cpu.addI(x)

		case 0x0029: // FX29: Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
			cpu.setI(x)

		case 0x0033: // FX33: Stores the Binary-coded decimal representation of VX at the addresses I, I plus 1, and I plus 2
			cpu.storeDecimalToI(mem, x)

		case 0x0055: // FX55: Stores V0 to VX in memory starting at address I
			cpu.storeToI(mem, x)

		case 0x0065: // FX65: Fills V0 to VX with values from memory starting at address I
			cpu.fetchFromI(mem, x)

		default:
			cpu.unknownOp(opc)
		}

	default:
		cpu.unknownOp(opc)
	}

	cpu.PC = newPC
}

func (cpu *CPU) jump(addr uint16) uint16 {
	return addr
}

func (cpu *CPU) call(addr uint16) uint16 {
	cpu.Stack[cpu.SP] = cpu.PC + 2
	cpu.SP++
	return addr
}

func (cpu *CPU) ret() uint16 {
	cpu.SP--
	return cpu.Stack[cpu.SP]
}

func (cpu *CPU) callRCA1802(addr uint16) {
	// TODO: call RCA 1802 program at address NNN
}

func (cpu *CPU) clearScreen(gfx *Graphics) {
	gfx.clear()
}

func (cpu *CPU) skipEq(x, val uint8) uint16 {
	if cpu.V[x] == val {
		return cpu.PC + 4 // skip next
	}
	return cpu.PC + 2
}

func (cpu *CPU) skipNeq(x, val uint8) uint16 {
	if cpu.V[x] != val {
		return cpu.PC + 4 // skip next
	}
	return cpu.PC + 2
}

func (cpu *CPU) SkipEqReg(x, y uint8) uint16 {
	if cpu.V[x] == cpu.V[y] {
		return cpu.PC + 4
	}
	return cpu.PC + 2
}

func (cpu *CPU) setV(x, val uint8) {
	cpu.V[x] = val
}

func (cpu *CPU) addV(x, val uint8) {
	cpu.V[x] += val
}

func (cpu *CPU) setXY(x, y uint8) {
	cpu.V[x] = cpu.V[y]
}

func (cpu *CPU) setXOrY(x, y uint8) {
	cpu.V[x] |= cpu.V[y]
}

func (cpu *CPU) setXAndY(x, y uint8) {
	cpu.V[x] &= cpu.V[y]
}

func (cpu *CPU) setXXorY(x, y uint8) {
	cpu.V[x] ^= cpu.V[y]
}

func (cpu *CPU) addXY(x, y uint8) {
	if cpu.V[x] > (0xFF - cpu.V[y]) {
		cpu.setCarry(1)
	} else {
		cpu.setCarry(0)
	}
	cpu.V[x] += cpu.V[y]
}

func (cpu *CPU) subXY(x, y uint8) {
	if cpu.V[x] > cpu.V[y] {
		cpu.setCarry(0)
	} else {
		cpu.setCarry(1)
	}
	cpu.V[x] -= cpu.V[y]
}

func (cpu *CPU) setCarry(carry uint8) {
	cpu.V[RegCarry] = carry
}

func (cpu *CPU) shiftRightX(x uint8) {
	cpu.setCarry(cpu.V[x] & 0x01)
	cpu.V[x] >>= 1
}

func (cpu *CPU) minusXY(x, y uint8) {
	if cpu.V[x] > cpu.V[y] {
		cpu.setCarry(0)
	} else {
		cpu.setCarry(1)
	}
	cpu.V[x] = cpu.V[y] - cpu.V[x]
}

func (cpu *CPU) shiftLeft(x uint8) {
	cpu.setCarry(cpu.V[x] >> 7)
	cpu.V[x] <<= 1
}

func (cpu *CPU) skipNeqReg(x, y uint8) uint16 {
	if cpu.V[x] != cpu.V[y] {
		return cpu.PC + 4
	}
	return cpu.PC + 2
}

func (cpu *CPU) mvi(index uint16) {
	cpu.I = index
}

func (cpu *CPU) jv0(addr uint16) uint16 {
	return addr + uint16(cpu.V[0])
}

func (cpu *CPU) rand(x, val uint8) {
	cpu.V[x] = uint8(rand.Uint32()&0xFF) & val
}

func (cpu *CPU) drawSprite(mem *Memory, gfx *Graphics, x, y, h uint8) {
	// Each row of 8 pixels is read as bit-coded starting from memory location I;
	// I value doesn't change after the execution of this instruction.
	// VF is set to 1 if any screen pixels are flipped from set to unset when the sprite is drawn,
	// and to 0 if that doesn't happen
	if hit := gfx.draw(mem, cpu.I, cpu.V[x], cpu.V[y], h); hit {
		cpu.setCarry(1)
	} else {
		cpu.setCarry(0)
	}
}

func (cpu *CPU) skipIfKeyPressed(sys *System, x uint8) uint16 {
	if sys.keys[cpu.V[x]] != 0 {
		return cpu.PC + 4
	}
	return cpu.PC + 2
}

func (cpu *CPU) skipIfKeyNotPressed(sys *System, x uint8) uint16 {
	if sys.keys[cpu.V[x]] == 0 {
		return cpu.PC + 4
	}
	return cpu.PC + 2
}

func (cpu *CPU) unknownOp(opc uint16) {
	fmt.Errorf("unknown opcode: %x\n", opc)
	cpu.PC += 2
}

func (cpu *CPU) getDelayTimer(sys *System, x uint8) {
	cpu.V[x] = sys.delayTimer
}

func (cpu *CPU) awaitKey(sys *System, x uint8) uint16 {
	fmt.Println("awaitKey")
	for i, key := range sys.keys {
		if key != 0 {
			cpu.V[x] = uint8(i)
			return cpu.PC + 2
		}
	}
	return cpu.PC // try again in next cycle
}

func (cpu *CPU) setDelayTimer(sys *System, x uint8) {
	sys.delayTimer = cpu.V[x]
}

func (cpu *CPU) setSoundTimer(sys *System, x uint8) {
	sys.soundTimer = cpu.V[x]
}

func (cpu *CPU) addI(x uint8) {
	addr := cpu.I + uint16(cpu.V[x])
	if addr > 0x0FFF {
		cpu.setCarry(1)
	} else {
		cpu.setCarry(0)
	}
	cpu.I = addr
}

func (cpu *CPU) setI(x uint8) {
	cpu.I = uint16(cpu.V[x]) * 5
}

func (cpu *CPU) storeDecimalToI(mem *Memory, x uint8) {
	mem[cpu.I] = cpu.V[x] / 100
	mem[cpu.I+1] = (cpu.V[x] / 10) % 10
	mem[cpu.I+2] = cpu.V[x] % 10
}

func (cpu *CPU) storeToI(mem *Memory, x uint8) {
	for i := uint8(0); i <= x; i++ {
		mem[cpu.I+uint16(i)] = cpu.V[i]
	}
}

func (cpu *CPU) fetchFromI(mem *Memory, x uint8) {
	for i := uint8(0); i <= x; i++ {
		cpu.V[i] = mem[cpu.I+uint16(i)]
	}
}
