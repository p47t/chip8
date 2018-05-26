package chip8

import (
	"fmt"
	"io/ioutil"
)

type System struct {
	cpu CPU
	mem Memory
	gfx Graphics

	delayTimer uint8
	soundTimer uint8
}

func (sys *System) Initialize() {
	sys.cpu.Reset()
	sys.mem.Clear()
	sys.gfx.clear()
}

func (sys *System) Cycle() {
	sys.cpu.Cycle(&sys.mem, &sys.gfx, sys)

	sys.updateTimer()
}

func (sys *System) updateTimer() {
	if sys.delayTimer > 0 {
		sys.delayTimer--
	}
	if sys.soundTimer > 0 {
		sys.soundTimer--
		if sys.soundTimer == 0 {
			sys.beep()
		}
	}
}

func (sys *System) beep() {
	fmt.Println("beep!")
}

func (sys *System) Load(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return sys.mem.loadROM(bytes)
}