package imui

import (
	_ "embed"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

//go:embed gl-shader/main.vert
var unversionedVertexShader string

//go:embed gl-shader/main.frag
var unversionedFragmentShader string

// IMUI implements a ui based on sdl/imgui
type IMUI struct {
	context *imgui.Context
	imguiIO imgui.IO

	window      *sdl.Window
	time        uint64
	buttonsDown [mouseButtonCount]bool

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
func NewIMUI(window *sdl.Window, font *imgui.FontAtlas) *IMUI {
	ui := &IMUI{
		context:     imgui.CreateContext(font),
		window:      window,
		glslVersion: "#version 150",
	}
	ui.imguiIO = imgui.CurrentIO()
	ui.imguiIO.SetClipboard(ui)
	ui.imguiIO.SetIniFilename("")

	ui.setKeyMapping()
	ui.createDeviceObjects()
	return ui
}

// Dispose cleans up the resources.
func (ui *IMUI) Dispose() {
	ui.invalidateDeviceObjects()
}

// NewFrame marks the begin of a render pass. It forwards all current state to imgui.CurrentIO().
func (ui *IMUI) NewFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	displayWidth, displayHeight := ui.window.GetSize()
	ui.imguiIO.SetDisplaySize(imgui.Vec2{X: float32(displayWidth), Y: float32(displayHeight)})

	// Setup time step (we don't use SDL_GetTicks() because it is using millisecond resolution)
	frequency := sdl.GetPerformanceFrequency()
	currentTime := sdl.GetPerformanceCounter()
	if ui.time > 0 {
		ui.imguiIO.SetDeltaTime(float32(currentTime-ui.time) / float32(frequency))
	} else {
		const fallbackDelta = 1.0 / 60.0
		ui.imguiIO.SetDeltaTime(fallbackDelta)
	}
	ui.time = currentTime

	// If a mouse press event came, always pass it as "mouse held this frame", so we don't miss click-release events that are shorter than 1 frame.
	x, y, state := sdl.GetMouseState()
	ui.imguiIO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	for i, button := range []uint32{sdl.BUTTON_LEFT, sdl.BUTTON_RIGHT, sdl.BUTTON_MIDDLE} {
		ui.imguiIO.SetMouseButtonDown(i, ui.buttonsDown[i] || (state&sdl.Button(button)) != 0)
		ui.buttonsDown[i] = false
	}

	imgui.NewFrame()
}

func (ui *IMUI) setKeyMapping() {
	keys := map[int]int{
		imgui.KeyTab:        sdl.SCANCODE_TAB,
		imgui.KeyLeftArrow:  sdl.SCANCODE_LEFT,
		imgui.KeyRightArrow: sdl.SCANCODE_RIGHT,
		imgui.KeyUpArrow:    sdl.SCANCODE_UP,
		imgui.KeyDownArrow:  sdl.SCANCODE_DOWN,
		imgui.KeyPageUp:     sdl.SCANCODE_PAGEUP,
		imgui.KeyPageDown:   sdl.SCANCODE_PAGEDOWN,
		imgui.KeyHome:       sdl.SCANCODE_HOME,
		imgui.KeyEnd:        sdl.SCANCODE_END,
		imgui.KeyInsert:     sdl.SCANCODE_INSERT,
		imgui.KeyDelete:     sdl.SCANCODE_DELETE,
		imgui.KeyBackspace:  sdl.SCANCODE_BACKSPACE,
		imgui.KeySpace:      sdl.SCANCODE_BACKSPACE,
		imgui.KeyEnter:      sdl.SCANCODE_RETURN,
		imgui.KeyEscape:     sdl.SCANCODE_ESCAPE,
		imgui.KeyA:          sdl.SCANCODE_A,
		imgui.KeyC:          sdl.SCANCODE_C,
		imgui.KeyV:          sdl.SCANCODE_V,
		imgui.KeyX:          sdl.SCANCODE_X,
		imgui.KeyY:          sdl.SCANCODE_Y,
		imgui.KeyZ:          sdl.SCANCODE_Z,
	}

	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	for imguiKey, nativeKey := range keys {
		ui.imguiIO.KeyMap(imguiKey, nativeKey)
	}
}

func (ui *IMUI) ProcessEvent(event sdl.Event) {
	switch event.GetType() {
	case sdl.MOUSEWHEEL:
		wheelEvent := event.(*sdl.MouseWheelEvent)
		var deltaX, deltaY float32
		if wheelEvent.X > 0 {
			deltaX++
		} else if wheelEvent.X < 0 {
			deltaX--
		}
		if wheelEvent.Y > 0 {
			deltaY++
		} else if wheelEvent.Y < 0 {
			deltaY--
		}
		ui.imguiIO.AddMouseWheelDelta(deltaX, deltaY)
	case sdl.MOUSEBUTTONDOWN:
		buttonEvent := event.(*sdl.MouseButtonEvent)
		switch buttonEvent.Button {
		case sdl.BUTTON_LEFT:
			ui.buttonsDown[mouseButtonPrimary] = true
		case sdl.BUTTON_RIGHT:
			ui.buttonsDown[mouseButtonSecondary] = true
		case sdl.BUTTON_MIDDLE:
			ui.buttonsDown[mouseButtonTertiary] = true
		}
	case sdl.TEXTINPUT:
		inputEvent := event.(*sdl.TextInputEvent)
		ui.imguiIO.AddInputCharacters(string(inputEvent.Text[:]))
	case sdl.KEYDOWN:
		keyEvent := event.(*sdl.KeyboardEvent)
		ui.imguiIO.KeyPress(int(keyEvent.Keysym.Scancode))
		ui.updateKeyModifier()
	case sdl.KEYUP:
		keyEvent := event.(*sdl.KeyboardEvent)
		ui.imguiIO.KeyRelease(int(keyEvent.Keysym.Scancode))
		ui.updateKeyModifier()
	}
}

func (ui *IMUI) updateKeyModifier() {
	modState := sdl.GetModState()
	mapModifier := func(lMask sdl.Keymod, lKey int, rMask sdl.Keymod, rKey int) (lResult int, rResult int) {
		if (modState & lMask) != 0 {
			lResult = lKey
		}
		if (modState & rMask) != 0 {
			rResult = rKey
		}
		return
	}
	ui.imguiIO.KeyShift(mapModifier(sdl.KMOD_LSHIFT, sdl.SCANCODE_LSHIFT, sdl.KMOD_RSHIFT, sdl.SCANCODE_RSHIFT))
	ui.imguiIO.KeyCtrl(mapModifier(sdl.KMOD_LCTRL, sdl.SCANCODE_LCTRL, sdl.KMOD_RCTRL, sdl.SCANCODE_RCTRL))
	ui.imguiIO.KeyAlt(mapModifier(sdl.KMOD_LALT, sdl.SCANCODE_LALT, sdl.KMOD_RALT, sdl.SCANCODE_RALT))
}

// Text returns the current clipboard text, if available.
func (ui *IMUI) Text() (string, error) {
	return sdl.GetClipboardText()
}

// SetText sets the text as the current clipboard text.
func (ui *IMUI) SetText(text string) {
	_ = sdl.SetClipboardText(text)
}

// Render translates the ImGui draw data to OpenGL commands.
func (ui *IMUI) Render() {
	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := ui.window.GetSize()
	fbWidth, fbHeight := ui.window.GLGetDrawableSize()
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
