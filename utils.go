package main

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	glExtensions = map[string]bool{}
)

// Initialize window and opengl context
func InitOpenglContext(title string, size, version []int) (*sdl.Window, error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return nil, err
	}

	// Create window and OpenGL context
	var (
		flags         = sdl.WINDOW_SHOWN | sdl.WINDOW_OPENGL
		width, height int
	)
	if size == nil {
		flags |= sdl.WINDOW_FULLSCREEN
	} else {
		width, height = size[0], size[1]
	}
	window, err := sdl.CreateWindow(
		title,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		int32(width),
		int32(height),
		uint32(flags))
	if err != nil {
		return nil, err
	}
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, version[0])
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, version[1])
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	_, err = window.GLCreateContext()
	if err != nil {
		return nil, err
	}
	err = sdl.GLSetSwapInterval(1)
	if err != nil {
		return nil, err
	}

	// Initialize OpenGL Bindings
	if err := gl.Init(); err != nil {
		return nil, err
	}

	glVersion := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Printf("OpenGL Version: %s\n", glVersion)

	var extNum int32
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &extNum)
	fmt.Printf("OpenGL Extensions: %d\n", extNum)
	for i := int32(0); i < extNum; i++ {
		extName := gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i)))
		glExtensions[extName] = true
		fmt.Printf("\t%s\n", extName)
	}

	return window, nil
}

type Shader struct {
	Source string
	Type   uint32
	id     uint32
}

func LoadShaders(ss []Shader) (uint32, error) {
	if len(ss) < 2 {
		return 0, errors.New("at least vertex and fragment shaders are needed")
	}

	for i := range ss {
		shaderID, err := compileShader(ss[i].Source, ss[i].Type)
		if err != nil {
			return 0, err
		}
		ss[i].id = shaderID
		defer gl.DeleteShader(shaderID)
	}

	program := gl.CreateProgram()
	for _, v := range ss {
		gl.AttachShader(program, v.id)
	}
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	return program, nil
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

func LoadTexture(file string) (uint32, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}
