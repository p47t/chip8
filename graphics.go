package chip8

type Graphics struct {
	buffer [64 * 32]uint8
	dirty bool
}

func (g *Graphics) isDirty() bool {
	return g.dirty
}

func (g *Graphics) clear() {
	for i := 0; i < len(g.buffer); i++ {
		g.buffer[i] = 0
	}
}

func (g *Graphics) getPixel(x, y uint8) uint8 {
	return g.buffer[x + y * 64]
}

func (g *Graphics) flip(x, y uint8) {
	g.buffer[x + y * 64] ^= 1
}

func (g *Graphics) draw(mem *Memory, I uint16, x, y, h uint8) bool {
	hit := false
	for r := uint8(0); r < h; r++ {
		pixel := mem[I + uint16(r)]
		for c := uint8(0); c < 8; c++ {
			if pixel & (0x80 >> c) != 0 {
				if g.getPixel(x + c, y + r) != 0 {
					hit = true
				}
				g.flip(x + c, y + r)
			}
		}
	}
	g.dirty = true
	return hit
}

