package glslfilter

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/go-gl/gl/v3.3-core/gl"
	"gopkg.in/yaml.v2"
)

type TextureFilterType int32

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
