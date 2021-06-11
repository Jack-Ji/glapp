package imui

import (
	_ "embed"
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/inkyblackness/imgui-go/v4"
)

//go:embed gl-shader/main.vert
var unversionedVertexShader string

//go:embed gl-shader/main.frag
var unversionedFragmentShader string

// IMUI implements a ui based on glfw/imgui
type IMUI struct {
	context *imgui.Context
	imguiIO imgui.IO

	window *glfw.Window

	time             float64
	mouseJustPressed [3]bool

	glslVersion            string
	fontTexture            uint32
	shaderHandle           uint32
	vertHandle             uint32
	fragHandle             uint32
	attribLocationTex      int32
	attribLocationProjMtx  int32
	attribLocationPosition int32
	attribLocationUV       int32
	attribLocationColor    int32
	vboHandle              uint32
	elementsHandle         uint32
}

// NewIMUI attempts to initialize a IMUI context.
func NewIMUI(window *glfw.Window, font *imgui.FontAtlas) (*IMUI, error) {
	ui := &IMUI{
		context:     imgui.CreateContext(font),
		window:      window,
		glslVersion: "#version 150",
	}
	ui.imguiIO = imgui.CurrentIO()
	ui.imguiIO.SetClipboard(ui)

	ui.setKeyMapping()
	ui.installCallbacks()
	ui.createDeviceObjects()

	return ui, nil
}

// NewFrame marks the begin of a render pass. It forwards all current state to imgui IO.
func (ui *IMUI) NewFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	w, h := ui.window.GetSize()
	ui.imguiIO.SetDisplaySize(imgui.Vec2{X: float32(w), Y: float32(h)})

	// Setup time step
	currentTime := glfw.GetTime()
	if ui.time > 0 {
		ui.imguiIO.SetDeltaTime(float32(currentTime - ui.time))
	}
	ui.time = currentTime

	// Setup inputs
	if ui.window.GetAttrib(glfw.Focused) != 0 {
		x, y := ui.window.GetCursorPos()
		ui.imguiIO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	} else {
		ui.imguiIO.SetMousePosition(imgui.Vec2{X: -math.MaxFloat32, Y: -math.MaxFloat32})
	}

	for i := 0; i < len(ui.mouseJustPressed); i++ {
		down := ui.mouseJustPressed[i] || (ui.window.GetMouseButton(glfwButtonIDByIndex[i]) == glfw.Press)
		ui.imguiIO.SetMouseButtonDown(i, down)
		ui.mouseJustPressed[i] = false
	}

	imgui.NewFrame()
}

func (ui *IMUI) setKeyMapping() {
	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	ui.imguiIO.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	ui.imguiIO.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	ui.imguiIO.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	ui.imguiIO.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	ui.imguiIO.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	ui.imguiIO.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	ui.imguiIO.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	ui.imguiIO.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	ui.imguiIO.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	ui.imguiIO.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	ui.imguiIO.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	ui.imguiIO.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	ui.imguiIO.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	ui.imguiIO.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	ui.imguiIO.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	ui.imguiIO.KeyMap(imgui.KeyA, int(glfw.KeyA))
	ui.imguiIO.KeyMap(imgui.KeyC, int(glfw.KeyC))
	ui.imguiIO.KeyMap(imgui.KeyV, int(glfw.KeyV))
	ui.imguiIO.KeyMap(imgui.KeyX, int(glfw.KeyX))
	ui.imguiIO.KeyMap(imgui.KeyY, int(glfw.KeyY))
	ui.imguiIO.KeyMap(imgui.KeyZ, int(glfw.KeyZ))
}

func (ui *IMUI) installCallbacks() {
	ui.window.SetMouseButtonCallback(ui.mouseButtonChange)
	ui.window.SetScrollCallback(ui.mouseScrollChange)
	ui.window.SetKeyCallback(ui.keyChange)
	ui.window.SetCharCallback(ui.charChange)
}

var glfwButtonIndexByID = map[glfw.MouseButton]int{
	glfw.MouseButton1: mouseButtonPrimary,
	glfw.MouseButton2: mouseButtonSecondary,
	glfw.MouseButton3: mouseButtonTertiary,
}

var glfwButtonIDByIndex = map[int]glfw.MouseButton{
	mouseButtonPrimary:   glfw.MouseButton1,
	mouseButtonSecondary: glfw.MouseButton2,
	mouseButtonTertiary:  glfw.MouseButton3,
}

func (ui *IMUI) mouseButtonChange(window *glfw.Window, rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	buttonIndex, known := glfwButtonIndexByID[rawButton]

	if known && (action == glfw.Press) {
		ui.mouseJustPressed[buttonIndex] = true
	}
}

func (ui *IMUI) mouseScrollChange(window *glfw.Window, x, y float64) {
	ui.imguiIO.AddMouseWheelDelta(float32(x), float32(y))
}

func (ui *IMUI) keyChange(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		ui.imguiIO.KeyPress(int(key))
	}
	if action == glfw.Release {
		ui.imguiIO.KeyRelease(int(key))
	}

	// Modifiers are not reliable across systems
	ui.imguiIO.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
	ui.imguiIO.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
	ui.imguiIO.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
	ui.imguiIO.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))
}

func (ui *IMUI) charChange(window *glfw.Window, char rune) {
	ui.imguiIO.AddInputCharacters(string(char))
}

// Text returns the current clipboard text, if available.
func (ui *IMUI) Text() (string, error) {
	return ui.window.GetClipboardString(), nil
}

// SetText sets the text as the current clipboard text.
func (ui *IMUI) SetText(text string) {
	ui.window.SetClipboardString(text)
}

// Dispose cleans up the resources.
func (ui *IMUI) Dispose() {
	ui.invalidateDeviceObjects()
}

// Render translates the ImGui draw data to OpenGL commands.
func (ui *IMUI) Render() {
	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := ui.window.GetSize()
	fbWidth, fbHeight := ui.window.GetFramebufferSize()
	if (fbWidth <= 0) || (fbHeight <= 0) {
		return
	}

	imgui.Render()
	drawData := imgui.RenderedDrawData()
	drawData.ScaleClipRects(imgui.Vec2{
		X: float32(fbWidth) / float32(displayWidth),
		Y: float32(fbHeight) / float32(displayHeight),
	})

	// Backup GL state
	var lastActiveTexture int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &lastActiveTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	var lastProgram int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &lastProgram)
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	var lastSampler int32
	gl.GetIntegerv(gl.SAMPLER_BINDING, &lastSampler)
	var lastArrayBuffer int32
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	var lastElementArrayBuffer int32
	gl.GetIntegerv(gl.ELEMENT_ARRAY_BUFFER_BINDING, &lastElementArrayBuffer)
	var lastVertexArray int32
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)
	var lastPolygonMode [2]int32
	gl.GetIntegerv(gl.POLYGON_MODE, &lastPolygonMode[0])
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])
	var lastScissorBox [4]int32
	gl.GetIntegerv(gl.SCISSOR_BOX, &lastScissorBox[0])
	var lastBlendSrcRgb int32
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &lastBlendSrcRgb)
	var lastBlendDstRgb int32
	gl.GetIntegerv(gl.BLEND_DST_RGB, &lastBlendDstRgb)
	var lastBlendSrcAlpha int32
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &lastBlendSrcAlpha)
	var lastBlendDstAlpha int32
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &lastBlendDstAlpha)
	var lastBlendEquationRgb int32
	gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &lastBlendEquationRgb)
	var lastBlendEquationAlpha int32
	gl.GetIntegerv(gl.BLEND_EQUATION_ALPHA, &lastBlendEquationAlpha)
	lastEnableBlend := gl.IsEnabled(gl.BLEND)
	lastEnableCullFace := gl.IsEnabled(gl.CULL_FACE)
	lastEnableDepthTest := gl.IsEnabled(gl.DEPTH_TEST)
	lastEnableScissorTest := gl.IsEnabled(gl.SCISSOR_TEST)

	// Setup render state: alpha-blending enabled, no face culling, no depth testing, scissor enabled, polygon fill
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Setup viewport, orthographic projection matrix
	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.
	gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
	orthoProjection := [4][4]float32{
		{2.0 / float32(displayWidth), 0.0, 0.0, 0.0},
		{0.0, 2.0 / -float32(displayHeight), 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}
	gl.UseProgram(ui.shaderHandle)
	gl.Uniform1i(ui.attribLocationTex, 0)
	gl.UniformMatrix4fv(ui.attribLocationProjMtx, 1, false, &orthoProjection[0][0])
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	var vaoHandle uint32
	gl.GenVertexArrays(1, &vaoHandle)
	gl.BindVertexArray(vaoHandle)
	gl.BindBuffer(gl.ARRAY_BUFFER, ui.vboHandle)
	gl.EnableVertexAttribArray(uint32(ui.attribLocationPosition))
	gl.EnableVertexAttribArray(uint32(ui.attribLocationUV))
	gl.EnableVertexAttribArray(uint32(ui.attribLocationColor))
	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointerWithOffset(uint32(ui.attribLocationPosition), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetPos))
	gl.VertexAttribPointerWithOffset(uint32(ui.attribLocationUV), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetUv))
	gl.VertexAttribPointerWithOffset(uint32(ui.attribLocationColor), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), uintptr(vertexOffsetCol))
	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	const bytesPerUint32 = 4
	if indexSize == bytesPerUint32 {
		drawType = gl.UNSIGNED_INT
	}

	// Draw
	for _, list := range drawData.CommandLists() {
		var indexBufferOffset uintptr

		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, ui.vboHandle)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.IndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ui.elementsHandle)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				gl.BindTexture(gl.TEXTURE_2D, uint32(cmd.TextureID()))
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))
				gl.DrawElementsWithOffset(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), indexBufferOffset)
			}
			indexBufferOffset += uintptr(cmd.ElementCount() * indexSize)
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)

	// Restore modified GL state
	gl.UseProgram(uint32(lastProgram))
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.BindSampler(0, uint32(lastSampler))
	gl.ActiveTexture(uint32(lastActiveTexture))
	gl.BindVertexArray(uint32(lastVertexArray))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(lastElementArrayBuffer))
	gl.BlendEquationSeparate(uint32(lastBlendEquationRgb), uint32(lastBlendEquationAlpha))
	gl.BlendFuncSeparate(uint32(lastBlendSrcRgb), uint32(lastBlendDstRgb), uint32(lastBlendSrcAlpha), uint32(lastBlendDstAlpha))
	if lastEnableBlend {
		gl.Enable(gl.BLEND)
	} else {
		gl.Disable(gl.BLEND)
	}
	if lastEnableCullFace {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	if lastEnableDepthTest {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	if lastEnableScissorTest {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.PolygonMode(gl.FRONT_AND_BACK, uint32(lastPolygonMode[0]))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.Scissor(lastScissorBox[0], lastScissorBox[1], lastScissorBox[2], lastScissorBox[3])
}

func (ui *IMUI) createDeviceObjects() {
	// Backup GL state
	var lastTexture int32
	var lastArrayBuffer int32
	var lastVertexArray int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)

	vertexShader := ui.glslVersion + "\n" + unversionedVertexShader
	fragmentShader := ui.glslVersion + "\n" + unversionedFragmentShader

	ui.shaderHandle = gl.CreateProgram()
	ui.vertHandle = gl.CreateShader(gl.VERTEX_SHADER)
	ui.fragHandle = gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source string) {
		csource, free := gl.Strs(source + "\x00")
		defer free()

		gl.ShaderSource(handle, 1, csource, nil)
	}

	glShaderSource(ui.vertHandle, vertexShader)
	glShaderSource(ui.fragHandle, fragmentShader)
	gl.CompileShader(ui.vertHandle)
	gl.CompileShader(ui.fragHandle)
	gl.AttachShader(ui.shaderHandle, ui.vertHandle)
	gl.AttachShader(ui.shaderHandle, ui.fragHandle)
	gl.LinkProgram(ui.shaderHandle)

	ui.attribLocationTex = gl.GetUniformLocation(ui.shaderHandle, gl.Str("Texture"+"\x00"))
	ui.attribLocationProjMtx = gl.GetUniformLocation(ui.shaderHandle, gl.Str("ProjMtx"+"\x00"))
	ui.attribLocationPosition = gl.GetAttribLocation(ui.shaderHandle, gl.Str("Position"+"\x00"))
	ui.attribLocationUV = gl.GetAttribLocation(ui.shaderHandle, gl.Str("UV"+"\x00"))
	ui.attribLocationColor = gl.GetAttribLocation(ui.shaderHandle, gl.Str("Color"+"\x00"))

	gl.GenBuffers(1, &ui.vboHandle)
	gl.GenBuffers(1, &ui.elementsHandle)

	ui.createFontsTexture()

	// Restore modified GL state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	gl.BindVertexArray(uint32(lastVertexArray))
}

func (ui *IMUI) createFontsTexture() {
	// Build texture atlas
	image := ui.imguiIO.Fonts().TextureDataAlpha8()

	// Upload texture to graphics system
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GenTextures(1, &ui.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, ui.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height),
		0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	// Store our identifier
	ui.imguiIO.Fonts().SetTextureID(imgui.TextureID(ui.fontTexture))

	// Restore state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
}

func (ui *IMUI) invalidateDeviceObjects() {
	if ui.vboHandle != 0 {
		gl.DeleteBuffers(1, &ui.vboHandle)
	}
	ui.vboHandle = 0
	if ui.elementsHandle != 0 {
		gl.DeleteBuffers(1, &ui.elementsHandle)
	}
	ui.elementsHandle = 0

	if (ui.shaderHandle != 0) && (ui.vertHandle != 0) {
		gl.DetachShader(ui.shaderHandle, ui.vertHandle)
	}
	if ui.vertHandle != 0 {
		gl.DeleteShader(ui.vertHandle)
	}
	ui.vertHandle = 0

	if (ui.shaderHandle != 0) && (ui.fragHandle != 0) {
		gl.DetachShader(ui.shaderHandle, ui.fragHandle)
	}
	if ui.fragHandle != 0 {
		gl.DeleteShader(ui.fragHandle)
	}
	ui.fragHandle = 0

	if ui.shaderHandle != 0 {
		gl.DeleteProgram(ui.shaderHandle)
	}
	ui.shaderHandle = 0

	if ui.fontTexture != 0 {
		gl.DeleteTextures(1, &ui.fontTexture)
		ui.imguiIO.Fonts().SetTextureID(0)
		ui.fontTexture = 0
	}
}
