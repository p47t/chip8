package main

import (
	"fmt"
	"os"
	"runtime"
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: chip8 romfile")
		return
	}

	var emu Emulator
	emu.Initialize(os.Args[1])
	defer emu.Terminate()
	emu.Loop()
}
