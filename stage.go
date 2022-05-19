package glslfilter

import (
	"fmt"
	"image"
	"reflect"
	"strings"

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
	// TODO: target for optimization if necessary; could linearize and copy over arrays as a single buffer op instead.
	for bindingName, uniform := range stage.uniforms {
		switch uniform.Value.(type) {
		case [][]float32:
			for i, v := range uniform.Value.([][]float32) {
				stage.bindUniform(bindingName, i, uniform.Type, v)
			}
		case [][]int32:
			for i, v := range uniform.Value.([][]int32) {
				stage.bindUniform(bindingName, i, uniform.Type, v)
			}
		case [][]uint32:
			for i, v := range uniform.Value.([][]uint32) {
				stage.bindUniform(bindingName, i, uniform.Type, v)
			}
		default:
			return fmt.Errorf("non-normalized uniform value")
		}
	}
	return nil
}

func (stage *FilterStage) bindUniform(baseBindingName string, index int, typ UniformType, v interface{}) error {
	bindingName := baseBindingName
	if index > 0 {
		bindingName = fmt.Sprintf("%s[%d]", bindingName, index)
	}

	location := gl.GetUniformLocation(stage.program, gl.Str(bindingName+"\x00"))
	if location == kGLLocationNotFound {
		return layoutNotFoundError("location", bindingName)
	}

	if err := setUniformComponent(location, typ.ScalarType, typ.VectorSize, v); err != nil {
		return err
	}
	return nil
}

func setUniformComponent(location int32, scalarType ScalarTypeName, vectorSize int, v interface{}) error {
	switch scalarType {
	case Float:
		switch vectorSize {
		case 0:
			gl.Uniform1fv(
				location,
				1,
				&v.([]float32)[0],
			)
		case 2:
			gl.Uniform2fv(
				location,
				1,
				&v.([]float32)[0],
			)
		case 3:
			gl.Uniform3fv(
				location,
				1,
				&v.([]float32)[0],
			)
		case 4:
			gl.Uniform4fv(
				location,
				1,
				&v.([]float32)[0],
			)
		}
	case Int:
		switch vectorSize {
		case 0:
			gl.Uniform1iv(
				location,
				1,
				&v.([]int32)[0],
			)
		case 2:
			gl.Uniform2iv(
				location,
				1,
				&v.([]int32)[0],
			)
		case 3:
			gl.Uniform3iv(
				location,
				1,
				&v.([]int32)[0],
			)
		case 4:
			gl.Uniform4iv(
				location,
				1,
				&v.([]int32)[0],
			)
		}
	case Uint:
		switch vectorSize {
		case 0:
			gl.Uniform1uiv(
				location,
				1,
				&v.([]uint32)[0],
			)
		case 2:
			gl.Uniform2uiv(
				location,
				1,
				&v.([]uint32)[0],
			)
		case 3:
			gl.Uniform3uiv(
				location,
				1,
				&v.([]uint32)[0],
			)
		case 4:
			gl.Uniform4uiv(
				location,
				1,
				&v.([]uint32)[0],
			)
		}

	default:
		return fmt.Errorf("invalid uniform type specified")
	}
	return nil
}

func normalizeUniformValue(definition UniformDefinition) (normed interface{}) {
	if definition.Type.IsArray {
		return normalizeArray(definition.Value, definition.Type.ScalarType)
	} else {
		return normalizeValue(definition.Value, definition.Type.ScalarType)
	}
}

func normalizeArray(v interface{}, typ ScalarTypeName) (normed interface{}) {
	vValue := reflect.ValueOf(v)
	util.Invariant(vValue.Kind() == reflect.Slice)

	length := vValue.Len()

	switch typ {
	case Float:
		var float2D [][]float32
		for i := 0; i < length; i++ {
			fmt.Println(vValue.Index(i).Interface())
			float2D = append(float2D, normToFloat(vValue.Index(i).Interface()))
		}
		normed = float2D
	case Int:
		var int2D [][]int32
		for i := 0; i < length; i++ {
			int2D = append(int2D, normToInt(vValue.Index(i).Interface()))
		}
		normed = int2D
	case Uint:
		var uint2D [][]uint32
		for i := 0; i < length; i++ {
			uint2D = append(uint2D, normToUint(vValue.Index(i).Interface()))
		}
		normed = uint2D
	}

	return normed
}

func normalizeValue(v interface{}, typ ScalarTypeName) (normed interface{}) {
	switch typ {
	case Float:
		return [][]float32{normToFloat(v)}
	case Int:
		return [][]int32{normToInt(v)}
	case Uint:
		return [][]uint32{normToUint(v)}
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
	case []interface{}:
		// special case from reflection
		interfaceSlice := v.([]interface{})
		f32Slice := []float32{}
		for _, v := range interfaceSlice {
			f32Slice = append(f32Slice, normToFloat(v)...)
		}
		return f32Slice
	}
	panicMsg := fmt.Sprintf("normToFloat: unexpected type %s in bagging area", reflect.TypeOf(v))
	panic(panicMsg)
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
	case []interface{}:
		// special case from reflection
		interfaceSlice := v.([]interface{})
		int32Slice := []int32{}
		for _, v := range interfaceSlice {
			int32Slice = append(int32Slice, normToInt(v)...)
		}
		return int32Slice
	}
	panicMsg := fmt.Sprintf("normToInt: unexpected type %s in bagging area", reflect.TypeOf(v))
	panic(panicMsg)
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
	case []interface{}:
		// special case from reflection
		interfaceSlice := v.([]interface{})
		uint32Slice := []uint32{}
		for _, v := range interfaceSlice {
			uint32Slice = append(uint32Slice, normToUint(v)...)
		}
		return uint32Slice
	}
	panicMsg := fmt.Sprintf("normToUint: unexpected type %s in bagging area", reflect.TypeOf(v))
	panic(panicMsg)
}
