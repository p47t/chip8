package chip8

import "testing"

func BenchmarkInvaders(b *testing.B) {
	benchmarkRom(b, "roms/invaders.c8", 10000)
}

func BenchmarkPong2(b *testing.B) {
	benchmarkRom(b, "roms/pong2.c8", 10000)
}

func benchmarkRom(b *testing.B, rom string, cycles int) {
	var sys System
	sys.Initialize()
	sys.Load(rom)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < cycles; i++ {
			sys.Cycle()
		}
	}
}
