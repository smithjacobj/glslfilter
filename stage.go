package glslfilter

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/smithjacobj/glslfilter/util"
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
		if uniformDefinition.Type.IsBuffer {
			uniform.Value = uniform.createUniformBufferObject(uniformDefinition)
		} else {
			uniform.Value = normalizeUniformValue(uniformDefinition)
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

	v := reflect.ValueOf(definition.Value)
	byts := linearize(v, definition.Type)
	gl.BufferData(uboName, len(byts), gl.Ptr(&byts[0]), gl.UNIFORM_BUFFER)

	return uboName
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
	return fmt.Errorf("location %s not found", bindingName)
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
		kind := reflect.TypeOf(uniform.Value).Kind()
		if uniform.Type.IsBuffer {
			if kind != reflect.Uint32 {
				return fmt.Errorf("UBO should be specified as a UBO block binding (uint32)")
			}

			block := gl.GetUniformBlockIndex(stage.program, gl.Str(bindingName+"\x00"))
			if block == gl.INVALID_INDEX {
				return locationNotFoundError(bindingName)
			}

			gl.UniformBlockBinding(stage.program, block, uniform.Value.(uint32))
		} else {
			if kind != reflect.Slice {
				return fmt.Errorf("primitives should be normalized to a slice")
			}

			location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
			if location == kGLLocationNotFound {
				return locationNotFoundError(bindingName)
			}

			switch uniform.Type.ScalarType {
			case Float:
				switch uniform.Type.VectorSize {
				case 0:
					gl.Uniform1fv(
						location,
						1,
						&uniform.Value.([]float32)[0],
					)
				case 2:
					gl.Uniform2fv(
						location,
						2,
						&uniform.Value.([]float32)[0],
					)
				case 3:
					gl.Uniform3fv(
						location,
						3,
						&uniform.Value.([]float32)[0],
					)
				case 4:
					gl.Uniform4fv(
						location,
						4,
						&uniform.Value.([]float32)[0],
					)
				}
			case Int:
				switch uniform.Type.VectorSize {
				case 0:
					gl.Uniform1iv(
						location,
						1,
						&uniform.Value.([]int32)[0],
					)
				case 2:
					gl.Uniform2iv(
						location,
						2,
						&uniform.Value.([]int32)[0],
					)
				case 3:
					gl.Uniform3iv(
						location,
						3,
						&uniform.Value.([]int32)[0],
					)
				case 4:
					gl.Uniform4iv(
						location,
						4,
						&uniform.Value.([]int32)[0],
					)
				}
			case Uint:
				switch uniform.Type.VectorSize {
				case 0:
					gl.Uniform1uiv(
						location,
						1,
						&uniform.Value.([]uint32)[0],
					)
				case 2:
					gl.Uniform2uiv(
						location,
						2,
						&uniform.Value.([]uint32)[0],
					)
				case 3:
					gl.Uniform3uiv(
						location,
						3,
						&uniform.Value.([]uint32)[0],
					)
				case 4:
					gl.Uniform4uiv(
						location,
						4,
						&uniform.Value.([]uint32)[0],
					)
				}

			default:
				return fmt.Errorf("invalid uniform type specified")
			}
		}
	}
	return nil
}

func normalizeUniformValue(definition UniformDefinition) (normed interface{}) {
	return normalizeValue(definition.Value, definition.Type.ScalarType)
}

func normalizeValue(v interface{}, typ ScalarTypeName) (normed interface{}) {
	switch typ {
	case Float:
		return normToFloat(v)
	case Int:
		return normToInt(v)
	case Uint:
		return normToUint(v)
	}
	panic("normalizeUniformValue: invalid type specified")
}

func normToFloat(v interface{}) (normed []float32) {
	switch v.(type) {
	case float32:
		return []float32{v.(float32)}
	case []float32:
		return v.([]float32)
	case float64:
		return []float32{float32(v.(float64))}
	case []float64:
		f64Slice := v.([]float64)
		f32Slice := []float32{}
		for _, v := range f64Slice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	case int:
		return []float32{float32(v.(int))}
	case []int:
		intSlice := v.([]int)
		f32Slice := []float32{}
		for _, v := range intSlice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	case uint:
		return []float32{float32(v.(uint))}
	case []uint:
		uintSlice := v.([]uint)
		f32Slice := []float32{}
		for _, v := range uintSlice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	}
	panic("normToFloat: unexpected type in bagging area")
}

func normToInt(v interface{}) (normed []int32) {
	switch v.(type) {
	case float32:
		return []int32{int32(v.(float32))}
	case []float32:
		f32Slice := v.([]float32)
		intSlice := []int32{}
		for _, v := range f32Slice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	case float64:
		return []int32{int32(v.(float64))}
	case []float64:
		f64Slice := v.([]float64)
		intSlice := []int32{}
		for _, v := range f64Slice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	case int:
		return []int32{int32(v.(int))}
	case []int:
		intSlice := v.([]int)
		int32Slice := []int32{}
		for _, v := range intSlice {
			int32Slice = append(int32Slice, int32(v))
		}
		return int32Slice
	case uint:
		return []int32{int32(v.(uint))}
	case []uint:
		uintSlice := v.([]uint)
		intSlice := []int32{}
		for _, v := range uintSlice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	}
	panic("normToInt: unexpected type in bagging area")
}

func normToUint(v interface{}) (normed []uint32) {
	switch v.(type) {
	case float32:
		return []uint32{uint32(v.(float32))}
	case []float32:
		f32Slice := v.([]float32)
		uintSlice := []uint32{}
		for _, v := range f32Slice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case float64:
		return []uint32{uint32(v.(float64))}
	case []float64:
		f64Slice := v.([]float64)
		uintSlice := []uint32{}
		for _, v := range f64Slice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case int:
		return []uint32{uint32(v.(int))}
	case []int:
		intSlice := v.([]int)
		uintSlice := []uint32{}
		for _, v := range intSlice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case uint:
		return []uint32{uint32(v.(uint))}
	case []uint:
		uintSlice := v.([]uint)
		uint32Slice := []uint32{}
		for _, v := range uintSlice {
			uint32Slice = append(uint32Slice, uint32(v))
		}
		return uint32Slice
	}
	panic("normToUint: unexpected type in bagging area")
}

func getScalarType(typ UniformType) (scalarType ScalarTypeName) {
	return typ.ScalarType
}

func linearize(v reflect.Value, typ UniformType) (buffer []byte) {
	if v.Kind() != reflect.Slice {
		panic("unexpected datatype for UBO")
	}

	length := v.Len()

	if length <= 0 {
		return []byte{}
	}

	for i := 0; i < length; i++ {
		vSub := reflect.ValueOf(v.Index(i).Interface())
		if vSub.Kind() == reflect.Slice {
			buffer = alignStd140(buffer, typ)
			buffer = append(buffer, linearize(vSub, typ)...)
		} else {
			byts := [4]byte{}
			switch typ.ScalarType {
			case Float:
				vFloat := math.Float32bits(normToFloat(vSub.Interface())[0])
				byts = *(*[4]byte)(unsafe.Pointer(&vFloat))
				buffer = append(buffer, byts[:]...)
			case Int:
				vInt := normToInt(vSub.Interface())[0]
				byts = *(*[4]byte)(unsafe.Pointer(&vInt))
				buffer = append(buffer, byts[:]...)
			case Uint:
				vUint := normToInt(vSub.Interface())[0]
				byts = *(*[4]byte)(unsafe.Pointer(&vUint))
				buffer = append(buffer, byts[:]...)
			}
		}
	}

	return buffer
}

func alignStd140(buffer []byte, typ UniformType) []byte {
	const kWordLen = 4
	const kVec2Alignment = kWordLen * 2
	const kVec34Alignment = kWordLen * 4

	util.Invariant(len(buffer)%kWordLen == 0) // all values are 32-bit aligned, flag this here

	neededPadding := 0
	switch typ.VectorSize {
	case 0:
	case 2:
		mod := len(buffer) % kVec2Alignment
		if mod == kVec2Alignment {
			neededPadding = 0
		} else {
			neededPadding = kVec2Alignment - mod
		}
	case 3:
		fallthrough
	case 4:
		mod := len(buffer) % kVec34Alignment
		if mod == kVec34Alignment {
			neededPadding = 0
		} else {
			neededPadding = kVec34Alignment - mod
		}
	default:
		panic("unexpected vector size")
	}
	for i := 0; i < neededPadding; i++ {
		buffer = append(buffer, 0)
	}
	return buffer
}
