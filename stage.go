package glslfilter

import (
	"fmt"
	"image"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type Texture struct {
	Data        *image.RGBA
	BindingName string
	Filter      int32
}

type Uniform struct {
	Type       UniformType
	Value      interface{}
	BufferName uint32
}

type FilterStage struct {
	program  uint32
	textures map[string]uint32
	uniforms map[string]Uniform
}

func NewFilterStage(fragmentShaderSource string, textures []Texture, uniformDefinitions []UniformDefinition) (stage *FilterStage, err error) {
	stage = new(FilterStage)
	stage.textures = make(map[string]uint32)

	stage.program, err = newProgram(fragmentShaderSource)
	if err != nil {
		return nil, err
	}

	if !hasEnoughTextureUnits(len(textures) + 1) {
		return nil, fmt.Errorf("more textures defined than available texture units")
	}

	for _, texture := range textures {
		textureName := createTexture(texture.Data, texture.Filter)
		stage.textures[texture.BindingName] = textureName
	}

	for _, uniformDefinition := range uniformDefinitions {
		stage.uniforms = append(stage.uniforms, Uniform{
			Type: uniformDefinition.Type,
		})
		if uniformDefinition.Type == Buffer {
			uniform.createUniformBufferObject(uniformDefinition)
		} else {
			uniform.Value = uniformDefinition.Value
		}
	}

	return stage, err
}

func createTexture(texture *image.RGBA, filter int32) (texName uint32) {
	width := texture.Rect.Dx()
	height := texture.Rect.Dy()

	if filter == 0 {
		filter = gl.LINEAR
	}

	gl.CreateTextures(gl.TEXTURE_2D, 1, &texName)
	gl.TextureStorage2D(texName, 1, gl.RGBA8, int32(width), int32(height))
	gl.TextureSubImage2D(texName, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(texture.Pix))
	gl.TextureParameteri(texName, gl.TEXTURE_MIN_FILTER, filter)
	gl.TextureParameteri(texName, gl.TEXTURE_MAG_FILTER, filter)
	gl.TextureParameteri(texName, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TextureParameteri(texName, gl.TEXTURE_WRAP_T, gl.REPEAT)

	return texName
}

func (uniform *Uniform) createUniformBufferObject(definition UniformDefinition) (uboName uint32) {
	gl.CreateBuffers(1, &uboName)
	gl.NamedBufferData(uboName)
}

func newProgram(fragmentShaderSource string) (name uint32, err error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
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

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (name uint32, err error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source + "\x00")
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

func locationNotFoundError(bindingName string) error {
	return fmt.Errorf("binding %s not found", bindingName)
}

func (stage *FilterStage) bindDefinitionTextures() error {
	for bindingName, texture := range stage.textures {
		location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
		if location == kGLLocationNotFound {
			return locationNotFoundError(bindingName)
		} else {
			var binding int32 = -1
			gl.GetUniformiv(stage.program, location, &binding)
			if binding == kGLLocationNotFound {
				return fmt.Errorf("binding %s not found", bindingName)
			} else {
				gl.BindTextureUnit(uint32(binding), texture)
			}
		}
	}
	return nil
}

func (stage *FilterStage) bindDefinitionUniforms() error {
	for bindingName, uniform := range stage.uniforms {
		location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
		if location == kGLLocationNotFound {
			return locationNotFoundError(bindingName)
		} else {
			switch uniform.(type) {
			case int:
			case float32:
			case float64:
			case []int:
			case []float32:
			case []float64:
			default:
				bufferSize := unsafe.Sizeof(uniform)
				gl.GetUniform
			}
		}
	}
}
