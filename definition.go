package glslfilter

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"gopkg.in/yaml.v2"
)

type TextureFilterType int32

type UniformType int

const (
	Invalid UniformType = iota
	Float
	FloatVec2
	FloatVec3
	FloatVec4
	Int
	IntVec2
	IntVec3
	IntVec4
	Uint
	UintVec2
	UintVec3
	UintVec4
	Buffer
)

type TextureDefinition struct {
	Path   string
	Name   string
	Filter TextureFilterType
}

type UniformDefinition struct {
	Name  string
	Type  UniformType
	Value interface{}
}

type Definition struct {
	Render struct {
		Width  int
		Height int
	}
	Stages []struct {
		FragmentShaderPath string `yaml:"fragmentShaderPath"`
		Textures           []TextureDefinition
		Uniforms           []UniformDefinition
	}
}

func LoadDefinitionFromFile(reader io.Reader) (definition Definition, err error) {
	definitionString, err := ioutil.ReadAll(reader)
	if err != nil {
		return definition, err
	}
	return LoadDefinition(definitionString)
}

func LoadDefinition(definitionString []byte) (definition Definition, err error) {
	err = yaml.Unmarshal(definitionString, &definition)
	if err != nil {
		return definition, err
	}

	log.Println(definition)
	return definition, nil
}

func (filterType *TextureFilterType) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var rawString string
	if err = unmarshal(&rawString); err != nil {
		return err
	}

	switch rawString {
	case "NEAREST":
		*filterType = gl.NEAREST
	case "LINEAR":
		fallthrough
	default:
		*filterType = TextureFilterType(gl.LINEAR)
	}

	return nil
}

func (uniformType *UniformType) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var rawString string
	if err = unmarshal(&rawString); err != nil {
		return err
	}
	rawString = strings.ToLower(rawString)

	switch rawString {
	case "float":
		*uniformType = Float
	case "floatvec2":
		*uniformType = FloatVec2
	case "floatvec3":
		*uniformType = FloatVec3
	case "floatvec4":
		*uniformType = FloatVec4
	case "int":
		*uniformType = Int
	case "intvec2":
		*uniformType = IntVec2
	case "intvec3":
		*uniformType = IntVec3
	case "intvec4":
		*uniformType = IntVec4
	case "uint":
		*uniformType = Uint
	case "uintvec2":
		*uniformType = UintVec2
	case "uintvec3":
		*uniformType = UintVec3
	case "uintvec4":
		*uniformType = UintVec4
	case "buffer":
		*uniformType = Buffer
	default:
		return fmt.Errorf("Invalid uniform type specified, options are `(int|uint|float)[Vec[2-4]]` or `buffer`")
	}

	return nil
}
