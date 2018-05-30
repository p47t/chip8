package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
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

	screenData            []byte
	window                *glfw.Window
	fullScreenTriangleVAO uint32
	bufferTexture         uint32
	shaderProgram         uint32
}

const vertexShader = `
#version 330

noperspective out vec2 TexCoord;

void main(void) {
    TexCoord.x = (gl_VertexID == 2)? 2.0: 0.0;
    TexCoord.y = (gl_VertexID == 1)? 2.0: 0.0;

	gl_Position = vec4(2.0 * TexCoord - 1.0, 0.0, 1.0);
}
`

const fragmentShader = `
#version 330

uniform sampler2D buffer;
noperspective in vec2 TexCoord;

out vec3 outColor;

void main(void) {
	outColor = texture(buffer, TexCoord).rgb;
}
`

func (emu *Emulator) Initialize(romFile string) {
	var err error
	if err = glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	// Create window
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	emu.window, err = glfw.CreateWindow(DisplayWidth, DisplayHeight, "Chip8", nil, nil)
	if err != nil {
		panic(err)
	}
	emu.window.MakeContextCurrent()

	emu.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		switch action {
		case glfw.Press:
			switch key {
			case glfw.Key1:
				emu.sys.OnKeyDown(0x1)
			case glfw.Key2:
				emu.sys.OnKeyDown(0x2)
			case glfw.Key3:
				emu.sys.OnKeyDown(0x3)
			case glfw.Key4:
				emu.sys.OnKeyDown(0xC)
			case glfw.KeyQ:
				emu.sys.OnKeyDown(0x4)
			case glfw.KeyW:
				emu.sys.OnKeyDown(0x5)
			case glfw.KeyE:
				emu.sys.OnKeyDown(0x6)
			case glfw.KeyR:
				emu.sys.OnKeyDown(0xD)
			case glfw.KeyA:
				emu.sys.OnKeyDown(0x7)
			case glfw.KeyS:
				emu.sys.OnKeyDown(0x8)
			case glfw.KeyD:
				emu.sys.OnKeyDown(0x9)
			case glfw.KeyF:
				emu.sys.OnKeyDown(0xE)
			case glfw.KeyZ:
				emu.sys.OnKeyDown(0xA)
			case glfw.KeyX:
				emu.sys.OnKeyDown(0x0)
			case glfw.KeyC:
				emu.sys.OnKeyDown(0xB)
			case glfw.KeyY:
				emu.sys.OnKeyDown(0xF)
			}
		case glfw.Release:
			switch key {
			case glfw.Key1:
				emu.sys.OnKeyUp(0x1)
			case glfw.Key2:
				emu.sys.OnKeyUp(0x2)
			case glfw.Key3:
				emu.sys.OnKeyUp(0x3)
			case glfw.Key4:
				emu.sys.OnKeyUp(0xC)
			case glfw.KeyQ:
				emu.sys.OnKeyUp(0x4)
			case glfw.KeyW:
				emu.sys.OnKeyUp(0x5)
			case glfw.KeyE:
				emu.sys.OnKeyUp(0x6)
			case glfw.KeyR:
				emu.sys.OnKeyUp(0xD)
			case glfw.KeyA:
				emu.sys.OnKeyUp(0x7)
			case glfw.KeyS:
				emu.sys.OnKeyUp(0x8)
			case glfw.KeyD:
				emu.sys.OnKeyUp(0x9)
			case glfw.KeyF:
				emu.sys.OnKeyUp(0xE)
			case glfw.KeyZ:
				emu.sys.OnKeyUp(0xA)
			case glfw.KeyX:
				emu.sys.OnKeyUp(0x0)
			case glfw.KeyC:
				emu.sys.OnKeyUp(0xB)
			case glfw.KeyY:
				emu.sys.OnKeyUp(0xF)
			}
		}
	})

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}
	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	gl.GenVertexArrays(1, &emu.fullScreenTriangleVAO)
	gl.BindVertexArray(emu.fullScreenTriangleVAO)

	var status int32

	emu.shaderProgram = gl.CreateProgram()

	vs := gl.CreateShader(gl.VERTEX_SHADER)
	defer gl.DeleteShader(vs)
	vsCode, vsCodeFree := gl.Strs(vertexShader)
	defer vsCodeFree()
	gl.ShaderSource(vs, 1, vsCode, nil)
	gl.CompileShader(vs)
	gl.GetShaderiv(vs, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		panic(fmt.Errorf("failed to compile vertex shader"))
	}
	gl.AttachShader(emu.shaderProgram, vs)
	defer gl.DetachShader(emu.shaderProgram, vs)

	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	defer gl.DeleteShader(fs)
	fsCode, fsCodeFree := gl.Strs(fragmentShader)
	defer fsCodeFree()
	gl.ShaderSource(fs, 1, fsCode, nil)
	gl.CompileShader(fs)
	gl.GetShaderiv(fs, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		panic(fmt.Errorf("failed to compile fragment shader"))
	}
	gl.AttachShader(emu.shaderProgram, fs)
	defer gl.DetachShader(emu.shaderProgram, fs)

	gl.LinkProgram(emu.shaderProgram)
	gl.GetProgramiv(emu.shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		panic(fmt.Errorf("failed to link shaderProgram"))
	}

	emu.screenData = make([]byte, ScreenWidth*ScreenHeight*3)
	for i := 0; i < len(emu.screenData); i++ {
		emu.screenData[i] = 0x80
	}

	gl.GenTextures(1, &emu.bufferTexture)
	gl.BindTexture(gl.TEXTURE_2D, emu.bufferTexture)

	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGB,
		ScreenWidth, ScreenHeight, 0,
		gl.RGB, gl.UNSIGNED_BYTE, unsafe.Pointer(&emu.screenData[0]))

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	bufferLoc := gl.GetUniformLocation(emu.shaderProgram, gl.Str("buffer"+"\x00"))
	gl.Uniform1i(bufferLoc, 0)

	gl.Disable(gl.DEPTH_TEST)
	gl.UseProgram(emu.shaderProgram)

	// Initialize system
	emu.sys.Initialize()
	emu.sys.Load(romFile)
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

	gl.BindVertexArray(emu.fullScreenTriangleVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
}

func (emu *Emulator) Loop() {
	emu.sys.Print(true)

	for !emu.window.ShouldClose() {
		emu.sys.Cycle()
		emu.sys.Print(true)

		if emu.sys.IsDirty() {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			emu.UpdateTexture()
			emu.window.SwapBuffers()
			emu.sys.SetDirty(false)
		}
		glfw.PollEvents()

		time.Sleep(1 * time.Millisecond)
	}
}

func (emu *Emulator) Terminate() {
	gl.DeleteVertexArrays(1, &emu.fullScreenTriangleVAO)
	gl.DeleteTextures(1, &emu.bufferTexture)
	gl.DeleteProgram(emu.shaderProgram)
	glfw.Terminate()
}

func main() {
	var emu Emulator
	emu.Initialize(os.Args[1])
	defer emu.Terminate()

	emu.Loop()
}
