package glslfilter

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

const kGLLocationNotFound = -1
const kViewportSizeBindingName = "outputResolution\x00"
const kPreviousResultBindingName = "previousResult\x00"

var screenTriangleVertices = []float32{
	-1, -3, 0, 0, 2,
	-1, 1, 0, 0, 0,
	3, 1, 0, 2, 0,
}

// this is reversed because the FBO renders opposite the window buffer
var fboTriangleVertices = []float32{
	-1, 3, 0, 0, 2,
	-1, -1, 0, 0, 0,
	3, -1, 0, 2, 0,
}

const vertexPositionOffset = 0
const vertexPositionSize = 3
const vertexUVOffset = vertexPositionSize * unsafe.Sizeof(float32(0))
const vertexUVSize = 2
const vertexStride = (vertexPositionSize + vertexUVSize) * unsafe.Sizeof(float32(0))

const vertexPositionLocation = 0
const vertexUVLocation = 1
const vertexShaderSource = `
#version 330 core
#extension GL_ARB_separate_shader_objects : enable

layout(location=0) in vec3 vertexPosition;
layout(location=1) in vec2 vertexTexCoord;
layout(location=0) out vec2 fragTexCoord;

void main() {
	// no transform, this is direct to screen space
	gl_Position = vec4(vertexPosition, 1.0);

	fragTexCoord = vertexTexCoord;
}
`

const lastResultToScreen = `
#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) in vec2 fragTexCoord;
layout(location = 1) uniform ivec2 outputResolution;
layout(binding = 0) uniform sampler2D previousResult;

layout(location = 0) out vec4 fragColor;

void main() {
	fragColor = texture(previousResult, fragTexCoord);	
}`

type interstageFBO struct {
	fboName     uint32
	textureName uint32
}

type Engine struct {
	debug          bool
	drawToScreen   bool
	viewportSize   struct{ x, y int }
	fboVAO         uint32
	screenVAO      uint32
	stages         []*FilterStage
	drawStage      *FilterStage
	interstageFBOs [2]interstageFBO
}

func NewEngine(viewportDimensions image.Rectangle, debug bool, drawToScreen bool) (engine *Engine, err error) {
	engine = new(Engine)
	engine.drawToScreen = drawToScreen

	err = gl.Init()
	if err != nil {
		return nil, err
	}

	engine.debug = debug
	if debug {
		gl.Enable(gl.DEBUG_OUTPUT)
		gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
			log.Printf("GL: %s", message)
		}, nil)
	}

	engine.viewportSize.x = viewportDimensions.Dx()
	engine.viewportSize.y = viewportDimensions.Dy()

	return engine, nil
}

func (engine *Engine) Init(stages []*FilterStage) (err error) {
	for i := range engine.interstageFBOs {
		targetFBOName, targetFBOTextureName, err := createFramebufferTarget()
		if err != nil {
			return err
		}
		engine.interstageFBOs[i] = interstageFBO{targetFBOName, targetFBOTextureName}
		log.Printf("created FBO %d rendering to texture %d", targetFBOName, targetFBOTextureName)
	}

	engine.screenVAO = createWindowBufferVAO(screenTriangleVertices)
	engine.fboVAO = createWindowBufferVAO(fboTriangleVertices)
	engine.stages = stages

	if engine.drawStage, err = NewFilterStage(lastResultToScreen, []Texture{}); err != nil {
		return err
	}

	gl.ClearColor(0, 0, 0, 1)

	return nil
}

func (engine *Engine) Render() error {
	locationNotFoundError := func(bindingName string) error { return fmt.Errorf("binding %s not found", bindingName) }

	for i, stage := range engine.stages {
		gl.UseProgram(stage.program)

		viewportSizeLocation := gl.GetUniformLocation(stage.program, gl.Str(kViewportSizeBindingName))
		if viewportSizeLocation != kGLLocationNotFound {
			gl.Uniform2i(viewportSizeLocation, int32(engine.viewportSize.x), int32(engine.viewportSize.y))
		}

		if i > 0 {
			previousFBO := engine.interstageFBOs[(i-1)%2]
			previousResultLocation := gl.GetUniformLocation(stage.program, gl.Str(kPreviousResultBindingName))
			if previousResultLocation == kGLLocationNotFound {
				return locationNotFoundError(kPreviousResultBindingName)
			} else {
				gl.BindTextureUnit(uint32(previousResultLocation), previousFBO.textureName)
			}
		}

		for bindingName, texture := range stage.textures {
			bindingLocation := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
			if bindingLocation == kGLLocationNotFound {
				return locationNotFoundError(bindingName)
			} else {
				var textureUnit int32 = -1
				gl.GetUniformiv(stage.program, bindingLocation, &textureUnit)
				if textureUnit == kGLLocationNotFound {
					return fmt.Errorf("binding %s not found", bindingName)
				} else {
					gl.BindTextureUnit(uint32(textureUnit), texture)
				}
			}
		}

		targetFBO := engine.interstageFBOs[i%2]
		gl.BindFramebuffer(gl.FRAMEBUFFER, targetFBO.fboName)
		gl.BindVertexArray(engine.fboVAO)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}

	// Why do we render to an FBO, then to the screen? So we can read the texture image from the
	// last FBO for export.
	if engine.drawToScreen {
		gl.UseProgram(engine.drawStage.program)

		viewportSizeLocation := gl.GetUniformLocation(engine.drawStage.program, gl.Str(kViewportSizeBindingName))
		if viewportSizeLocation != kGLLocationNotFound {
			gl.Uniform2i(viewportSizeLocation, int32(engine.viewportSize.x), int32(engine.viewportSize.y))
		}

		previousFBOtexture := engine.getFinalResultTexture()
		previousResultLocation := gl.GetUniformLocation(engine.drawStage.program, gl.Str(kPreviousResultBindingName))
		if previousResultLocation == kGLLocationNotFound {
			return locationNotFoundError(kPreviousResultBindingName)
		} else {
			gl.BindTextureUnit(uint32(previousResultLocation), previousFBOtexture)
		}

		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.BindVertexArray(engine.screenVAO)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}

	return nil
}

func (engine *Engine) GetLastRenderImage() *image.RGBA {
	rect := image.Rect(0, 0, engine.viewportSize.x, engine.viewportSize.y)
	image := image.NewRGBA(rect)

	gl.GetTextureImage(engine.getFinalResultTexture(), 0, gl.RGBA, gl.UNSIGNED_BYTE, int32(len(image.Pix)), gl.Ptr(&image.Pix[0]))
	return image
}

func (engine *Engine) getFinalResultTexture() (texName uint32) {
	return engine.interstageFBOs[(len(engine.stages)-1)%2].textureName
}

func createWindowBufferVAO(vertices []float32) (name uint32) {
	var vbo uint32
	gl.CreateBuffers(1, &vbo)
	gl.NamedBufferStorage(vbo, len(vertices)*int(unsafe.Sizeof(float32(0))), gl.Ptr(vertices), gl.MAP_READ_BIT)

	var vao uint32
	gl.CreateVertexArrays(1, &vao)
	gl.EnableVertexArrayAttrib(vao, vertexPositionLocation)
	gl.VertexArrayVertexBuffer(vao, 0, vbo, vertexPositionOffset, int32(vertexStride))
	gl.VertexArrayAttribBinding(vao, vertexPositionLocation, 0)
	gl.VertexArrayAttribFormat(vao, vertexPositionLocation, vertexPositionSize, gl.FLOAT, false, uint32(vertexPositionOffset))

	gl.EnableVertexArrayAttrib(vao, vertexUVLocation)
	gl.VertexArrayVertexBuffer(vao, 1, vbo, int(vertexUVOffset), int32(vertexStride))
	gl.VertexArrayAttribBinding(vao, vertexUVLocation, 0)
	gl.VertexArrayAttribFormat(vao, vertexUVLocation, vertexUVSize, gl.FLOAT, false, uint32(vertexUVOffset))

	return vao
}

func createFramebufferTarget() (fboName, texName uint32, err error) {
	var viewBoundsVector [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewBoundsVector[0])
	viewBounds := image.Rect(int(viewBoundsVector[0]), int(viewBoundsVector[1]), int(viewBoundsVector[2]), int(viewBoundsVector[3]))
	gl.CreateFramebuffers(1, &fboName)
	gl.CreateTextures(gl.TEXTURE_2D, 1, &texName)
	gl.TextureStorage2D(texName, 1, gl.RGBA8, int32(viewBounds.Dx()), int32(viewBounds.Dy()))
	gl.TextureParameteri(texName, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TextureParameteri(texName, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.NamedFramebufferTexture(fboName, gl.COLOR_ATTACHMENT0, texName, 0)
	gl.NamedFramebufferDrawBuffer(fboName, gl.COLOR_ATTACHMENT0)

	status := gl.CheckNamedFramebufferStatus(fboName, gl.FRAMEBUFFER)
	if status != gl.FRAMEBUFFER_COMPLETE {
		return 0, 0, fmt.Errorf("error creating framebuffer: %d", status)
	}

	return fboName, texName, nil
}

func hasEnoughTextureUnits(n int) bool {
	var availableCount int32
	gl.GetIntegerv(gl.MAX_TEXTURE_IMAGE_UNITS, &availableCount)
	return n <= int(availableCount)
}

func LoadFragmentShader(fragmentShaderPath string) (string, error) {
	fragmentShaderFile, err := os.Open(fragmentShaderPath)
	if err != nil {
		return "", err
	}

	fragmentShaderSource, err := ioutil.ReadAll(fragmentShaderFile)
	if err != nil {
		return "", err
	}

	return string(fragmentShaderSource), nil
}

func LoadTextureData(path string) (texture *image.RGBA, err error) {
	log.Printf("loading texture: %s\n", path)
	imageFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	imageData, _, err := image.Decode(imageFile)
	if err != nil {
		return nil, err
	}

	imageRGBA := image.NewRGBA(imageData.Bounds())
	// we don't support cropped images (where the buffer size is larger than the image)
	if imageRGBA.Stride != imageRGBA.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride: %d", imageRGBA.Rect.Size().X*4)
	}
	draw.Draw(imageRGBA, imageRGBA.Bounds(), imageData, image.Point{0, 0}, draw.Src)

	return imageRGBA, nil
}
