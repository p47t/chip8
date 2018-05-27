package chip8

import (
	"fmt"
	"io/ioutil"

	tm "github.com/buger/goterm"
)

type System struct {
	cpu CPU
	mem Memory
	gfx Graphics

	keys [16]uint8

	delayTimer uint8
	soundTimer uint8
}

func (sys *System) Initialize() {
	sys.cpu.reset()
	sys.mem.clear()
	sys.gfx.clear()

	for i := 0; i < len(sys.keys); i++ {
		sys.keys[i] = 0
	}

	sys.delayTimer = 0
	sys.soundTimer = 0
}

func (sys *System) Print() {
	tm.Clear()
	tm.MoveCursor(1,1)

	sys.cpu.Print(tm.Screen)

	tm.Flush()
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

func (sys *System) GetPixel(x, y uint8) uint8 {
	return sys.gfx.getPixel(x, y)
}

func (sys *System) IsDirty() bool {
	return sys.gfx.isDirty()
}

func (sys *System) SetDirty(dirty bool) {
	sys.gfx.setDirty(dirty)
}
