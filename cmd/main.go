package main

import (
	"github.com/p47r1ck7541/chip8"
	"os"
)

func main() {
	var sys chip8.System
	sys.Initialize()
	sys.Load(os.Args[1])

	for {
		sys.Cycle()
	}
}
