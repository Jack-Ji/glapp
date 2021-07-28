package main

import (
	"fmt"
	_ "image/png"
	"log"
	"runtime"

	"glapp/iu"
	"glapp/iu/demo"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	// GUI event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	const (
		windowWidth  = 1280
		windowHeight = 800
		majorVersion = 4
		minorVersion = 6
	)

	window, err := initOpenglContext(
		"glapp",
		[]int{windowWidth, windowHeight},
		[]int{majorVersion, minorVersion})
	if err != nil {
		log.Fatal("Initialize OpenGL context failed:", err)
	}

	iuContext := iu.NewContext(window, nil, true)
	defer iuContext.Dispose()

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL Version: %s", version)

	// Configure the vertex and fragment shaders
	program, err := loadShader(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}
	gl.UseProgram(program)

	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/windowHeight, 0.1, 10.0)
	projectionUniform := gl.GetUniformLocation(program, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

	camera := mgl32.LookAtV(mgl32.Vec3{3, 3, 3}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
	cameraUniform := gl.GetUniformLocation(program, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

	model := mgl32.Ident4()
	modelUniform := gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

	textureUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

	// Load the texture
	texture, err := loadTexture("square.png")
	if err != nil {
		log.Fatalln(err)
	}

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 5*4, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, 5*4, 3*4)

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(1.0, 1.0, 1.0, 1.0)

	var (
		previousTime      = sdl.GetTicks()
		running           = true
		angle             = 0.0
		showDemoWindow    = false
		showGoDemoWindow  = false
		clearColor        = [3]float32{0.0, 0.0, 0.0}
		f                 = float32(0)
		counter           = 0
		showAnotherWindow = false
	)

	for running {
	EVENT_LOOP:
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			iuContext.ProcessEvent(event)

			switch event.(type) {
			case *sdl.QuitEvent:
				log.Printf("Quit")
				running = false
				break EVENT_LOOP
			}
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// 3d scene
		{
			time := sdl.GetTicks()
			elapsed := time - previousTime
			previousTime = time
			angle += float64(elapsed) / 1000
			model = mgl32.HomogRotate3D(float32(angle), mgl32.Vec3{0, 1, 0})

			// Render
			gl.UseProgram(program)
			gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
			gl.BindVertexArray(vao)

			gl.ActiveTexture(gl.TEXTURE0)
			gl.BindTexture(gl.TEXTURE_2D, texture)
			gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)
		}

		// ui rendering
		{
			iuContext.NewFrame()

			// 1. Show a simple window.
			// Tip: if we don't call imgui.Begin()/imgui.End() the widgets automatically appears in a window called "Debug".
			{
				imgui.Text("ภาษาไทย测试조선말")                   // To display these, you'll need to register a compatible font
				imgui.Text("Hello, world!")                  // Display some text
				imgui.SliderFloat("float", &f, 0.0, 1.0)     // Edit 1 float using a slider from 0.0f to 1.0f
				imgui.ColorEdit3("clear color", &clearColor) // Edit 3 floats representing a color

				imgui.Checkbox("Demo Window", &showDemoWindow) // Edit bools storing our window open/close state
				imgui.Checkbox("Go Demo Window", &showGoDemoWindow)
				imgui.Checkbox("Another Window", &showAnotherWindow)

				if imgui.Button("Button") { // Buttons return true when clicked (most widgets return true when edited/activated)
					counter++
				}
				imgui.SameLine()
				imgui.Text(fmt.Sprintf("counter = %d", counter))

				imgui.Text(fmt.Sprintf("Application average %.3f ms/frame (%.1f FPS)",
					1000/imgui.CurrentIO().Framerate(), imgui.CurrentIO().Framerate()))
			}

			// 2. Show another simple window. In most cases you will use an explicit Begin/End pair to name your windows.
			if showAnotherWindow {
				// Pass a pointer to our bool variable (the window will have a closing button that will clear the bool when clicked)
				imgui.BeginV("Another window", &showAnotherWindow, 0)
				imgui.Text("Hello from another window!")
				if imgui.Button("Close Me") {
					showAnotherWindow = false
				}
				imgui.End()
			}

			// 3. Show the ImGui demo window. Most of the sample code is in imgui.ShowDemoWindow().
			// Read its code to learn more about Dear ImGui!
			if showDemoWindow {
				// Normally user code doesn't need/want to call this because positions are saved in .ini file anyway.
				// Here we just want to make the demo initial state a bit more friendly!
				const demoX = 650
				const demoY = 20
				imgui.SetNextWindowPosV(imgui.Vec2{X: demoX, Y: demoY}, imgui.ConditionFirstUseEver, imgui.Vec2{})

				imgui.ShowDemoWindow(&showDemoWindow)
			}
			if showGoDemoWindow {
				demo.Show(&showGoDemoWindow)
			}

			iuContext.Render()
		}

		// Maintenance
		window.GLSwap()
	}
}

var vertexShader = `
#version 460 core
uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
in vec3 vert;
in vec2 vertTexCoord;
out vec2 fragTexCoord;
void main() {
    fragTexCoord = vertTexCoord;
    gl_Position = projection * camera * model * vec4(vert, 1);
}
` + "\x00"

var fragmentShader = `
#version 460 core
uniform sampler2D tex;
in vec2 fragTexCoord;
out vec4 outputColor;
void main() {
    outputColor = texture(tex, fragTexCoord);
}
` + "\x00"

var cubeVertices = []float32{
	//  X, Y, Z, U, V
	// Bottom
	-1.0, -1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,

	// Top
	-1.0, 1.0, -1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, 1.0, 1.0, 1.0,

	// Front
	-1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, -1.0, 1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,
	1.0, -1.0, 1.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,

	// Back
	-1.0, -1.0, -1.0, 0.0, 0.0,
	-1.0, 1.0, -1.0, 0.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 1.0,

	// Left
	-1.0, -1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, -1.0, 0.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0, 1.0, 0.0,

	// Right
	1.0, -1.0, 1.0, 1.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
	1.0, 1.0, -1.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 0.0, 1.0,
}
