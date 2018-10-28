package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/p47t/chip8"
)

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

var keyMap = map[glfw.Key]int{
	glfw.Key1: 0x1,
	glfw.Key2: 0x2,
	glfw.Key3: 0x3,
	glfw.Key4: 0xC,
	glfw.KeyQ: 0x4,
	glfw.KeyW: 0x5,
	glfw.KeyE: 0x6,
	glfw.KeyR: 0xD,
	glfw.KeyA: 0x7,
	glfw.KeyS: 0x8,
	glfw.KeyD: 0x9,
	glfw.KeyF: 0xE,
	glfw.KeyZ: 0xA,
	glfw.KeyX: 0x0,
	glfw.KeyC: 0xB,
	glfw.KeyY: 0xF,
}

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

	// Key handling
	emu.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		switch action {
		case glfw.Press:
			if c8Key, ok := keyMap[key]; ok {
				emu.sys.OnKeyDown(c8Key)
			}
		case glfw.Release:
			if c8Key, ok := keyMap[key]; ok {
				emu.sys.OnKeyUp(c8Key)
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

	vs, err := compileShader(vertexShader, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	defer gl.DeleteShader(vs)
	gl.AttachShader(emu.shaderProgram, vs)
	defer gl.DetachShader(emu.shaderProgram, vs)

	fs, err := compileShader(fragmentShader, gl.FRAGMENT_SHADER)
	defer gl.DeleteShader(fs)
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

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func (emu *Emulator) UpdateTexture() {
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			offset := ((ScreenHeight-y-1)*ScreenWidth + x) * 3
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
	for !emu.window.ShouldClose() {
		start := time.Now()

		glfw.PollEvents()

		for i := 0; i < chip8.SystemHz/chip8.TimerHz; i++ {
			emu.sys.Cycle()
			// emu.sys.Print(true)
		}

		if emu.sys.IsDirty() {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			emu.UpdateTexture()
			emu.window.SwapBuffers()

			emu.sys.SetDirty(false)
		}

		if elapsed, slice := time.Since(start), time.Second/chip8.TimerHz; elapsed < slice {
			time.Sleep(slice - elapsed)
		}

		emu.sys.UpdateTimer()
	}
}

func (emu *Emulator) Terminate() {
	gl.DeleteVertexArrays(1, &emu.fullScreenTriangleVAO)
	gl.DeleteTextures(1, &emu.bufferTexture)
	gl.DeleteProgram(emu.shaderProgram)
	glfw.Terminate()
}
