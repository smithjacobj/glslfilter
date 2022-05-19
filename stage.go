package glslfilter

import (
	"fmt"
	"image"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type Texture struct {
	Data        *image.RGBA
	BindingName string
	Filter      int32
}

type Uniform struct {
	Type  UniformType
	Value interface{}
}

type FilterStage struct {
	program  uint32
	textures map[string]uint32
	uniforms map[string]*Uniform
}

func NewFilterStage(fragmentShaderSource string, textures []Texture, uniformDefinitions []UniformDefinition) (stage *FilterStage, err error) {
	stage = new(FilterStage)
	stage.textures = make(map[string]uint32)
	stage.uniforms = make(map[string]*Uniform)

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
		uniform := Uniform{
			Type: uniformDefinition.Type,
		}
		stage.uniforms[uniformDefinition.Name] = &uniform
		uniform.Value = normalizeUniformValue(uniformDefinition)
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

func layoutNotFoundError(layoutName string, bindingName string) error {
	return fmt.Errorf("location for %s not found", bindingName)
}

func (stage *FilterStage) bindDefinitionTextures() error {
	for bindingName, texture := range stage.textures {
		location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
		if location == kGLLocationNotFound {
			return layoutNotFoundError("location", bindingName)
		} else {
			var binding int32 = kGLLocationNotFound
			gl.GetUniformiv(stage.program, location, &binding)
			if binding == kGLLocationNotFound {
				return layoutNotFoundError("binding", bindingName)
			} else {
				gl.BindTextureUnit(uint32(binding), texture)
			}
		}
	}
	return nil
}

func (stage *FilterStage) bindDefinitionUniforms() error {
	for bindingName, uniform := range stage.uniforms {
		switch v := uniform.Value.(type) {
		case []float32:
			stage.bindUniform(bindingName, len(v), uniform.Type, v)
		case []int32:
			stage.bindUniform(bindingName, len(v), uniform.Type, v)
		case []uint32:
			stage.bindUniform(bindingName, len(v), uniform.Type, v)
		default:
			return fmt.Errorf("non-normalized uniform value")
		}
	}
	return nil
}

func (stage *FilterStage) bindUniform(bindingName string, length int, typ UniformType, v interface{}) error {
	location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
	if location == kGLLocationNotFound {
		return layoutNotFoundError("location", bindingName)
	}

	if err := setUniform(location, int32(length), typ.ScalarType, typ.VectorSize, v); err != nil {
		return err
	}
	return nil
}

func setUniform(location int32, length int32, scalarType ScalarTypeName, vectorSize int, v interface{}) error {
	switch scalarType {
	case Float:
		switch vectorSize {
		case 0:
			gl.Uniform1fv(
				location,
				length,
				&v.([]float32)[0],
			)
		case 2:
			gl.Uniform2fv(
				location,
				length,
				&v.([]float32)[0],
			)
		case 3:
			gl.Uniform3fv(
				location,
				length,
				&v.([]float32)[0],
			)
		case 4:
			gl.Uniform4fv(
				location,
				length,
				&v.([]float32)[0],
			)
		}
	case Int:
		switch vectorSize {
		case 0:
			gl.Uniform1iv(
				location,
				length,
				&v.([]int32)[0],
			)
		case 2:
			gl.Uniform2iv(
				location,
				length,
				&v.([]int32)[0],
			)
		case 3:
			gl.Uniform3iv(
				location,
				length,
				&v.([]int32)[0],
			)
		case 4:
			gl.Uniform4iv(
				location,
				length,
				&v.([]int32)[0],
			)
		}
	case Uint:
		switch vectorSize {
		case 0:
			gl.Uniform1uiv(
				location,
				length,
				&v.([]uint32)[0],
			)
		case 2:
			gl.Uniform2uiv(
				location,
				length,
				&v.([]uint32)[0],
			)
		case 3:
			gl.Uniform3uiv(
				location,
				length,
				&v.([]uint32)[0],
			)
		case 4:
			gl.Uniform4uiv(
				location,
				length,
				&v.([]uint32)[0],
			)
		}

	default:
		return fmt.Errorf("invalid uniform type specified")
	}
	return nil
}
