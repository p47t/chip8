package chip8

const (
	GfxWidth      = 64
	GfxWidthBytes = GfxWidth / 8
	GfxHeight     = 32
)

type Graphics struct {
	buffer [GfxWidthBytes * GfxHeight]uint8
	dirty  bool
}

func (g *Graphics) isDirty() bool {
	return g.dirty
}

func (g *Graphics) setDirty(dirty bool) {
	g.dirty = dirty
}

func (g *Graphics) clear() {
	for i := 0; i < len(g.buffer); i++ {
		g.buffer[i] = 0
	}
	g.dirty = true
}

func (g *Graphics) getPixel(x, y uint8) uint8 {
	bit := 7 - (x % 8) // bit 7 is the first pixel and so on
	return g.buffer[uint(x)/8+uint(y)*GfxWidthBytes] & (1 << bit)
}

func (g *Graphics) draw(mem *Memory, I uint16, x, y, h uint8) bool {
	hit := false
	bit := x % 8
	for r := uint8(0); r < h; r++ {
		pixel := mem[I+uint16(r)]
		offset := uint(x)/8 + uint(y+r)*GfxWidthBytes
		if bit == 0 {
			// process 8 pixels in 8-bit operation
			if g.buffer[offset]&pixel != 0 {
				hit = true
			}
			g.buffer[offset] ^= pixel
		} else {
			// process 8 pixels in 16-bit operation
			dst := uint16(g.buffer[offset+1])<<8 | uint16(g.buffer[offset])
			src := uint16(pixel) << bit
			if src&dst != 0 {
				hit = true
			}
			r := dst ^ src
			g.buffer[offset] = uint8(r & 0xFF)
			g.buffer[offset+1] = uint8(r >> 8)
		}
	}
	g.dirty = true
	return hit
}
