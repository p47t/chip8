package main

import (
	"log"
	"os"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/p47r1ck7541/chip8"
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

const (
	ScreenWidth   = chip8.GfxWidth
	ScreenHeight  = chip8.GfxHeight
	DisplayScale  = 10
	DisplayWidth  = ScreenWidth * DisplayScale
	DisplayHeight = ScreenHeight * DisplayScale
)

type Emulator struct {
	sys chip8.System

	screenData []byte
	window     *glfw.Window
}

func (emu *Emulator) Initialize(romFile string) {
	var err error
	if err = glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	// Create window
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	emu.window, err = glfw.CreateWindow(DisplayWidth, DisplayHeight, "Chip8", nil, nil)
	if err != nil {
		panic(err)
	}
	emu.window.MakeContextCurrent()

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}
	gl.ClearColor(0.0, 0.5, 0.0, 0.0)
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, DisplayWidth, DisplayHeight, 0, -1.0, 1.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.Viewport(0, 0, DisplayWidth*2, DisplayHeight*2)

	emu.screenData = make([]byte, ScreenWidth*ScreenHeight*3)
	for i := 0; i < len(emu.screenData); i++ {
		emu.screenData[i] = 0x80
	}
	emu.SetupTexture()

	// Initialize system
	emu.sys.Initialize()
	emu.sys.Load(romFile)
}

func (emu *Emulator) SetupTexture() {
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGB, ScreenWidth, ScreenHeight, 0,
		gl.RGB, gl.UNSIGNED_BYTE, unsafe.Pointer(&emu.screenData[0]))

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP)

	gl.Enable(gl.TEXTURE_2D)
}

func (emu *Emulator) UpdateTexture() {
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			offset := (y*ScreenWidth + x) * 3
			if emu.sys.GetPixel(uint8(x), uint8(y)) == 0 {
				emu.screenData[offset], emu.screenData[offset+1], emu.screenData[offset+2] = 0, 0, 0
			} else {
				emu.screenData[offset], emu.screenData[offset+1], emu.screenData[offset+2] = 0xFF, 0xFF, 0xFF
			}
		}
	}

	gl.TexSubImage2D(
		gl.TEXTURE_2D, 0, 0, 0,
		ScreenWidth, ScreenHeight, gl.RGB, gl.UNSIGNED_BYTE,
		unsafe.Pointer(&emu.screenData[0]))

	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0.0, 0.0)
	gl.Vertex2d(0.0,0.0)
	gl.TexCoord2d(1.0, 0.0)
	gl.Vertex2d(DisplayWidth, 0.0)
	gl.TexCoord2d(1.0, 1.0)
	gl.Vertex2d(DisplayWidth, DisplayHeight)
	gl.TexCoord2d(0.0, 1.0)
	gl.Vertex2d(0.0, DisplayHeight)
	gl.End()
}

func (emu *Emulator) Loop() {
	for !emu.window.ShouldClose() {
		emu.sys.Cycle()
		emu.sys.Print()

		if emu.sys.IsDirty() {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			emu.UpdateTexture()
			emu.window.SwapBuffers()
			emu.sys.SetDirty(false)
		}
		glfw.PollEvents()
	}
}

func (emu *Emulator) Terminate() {
	glfw.Terminate()
}

func main() {
	var emu Emulator
	emu.Initialize(os.Args[1])
	defer emu.Terminate()

	emu.Loop()
}
