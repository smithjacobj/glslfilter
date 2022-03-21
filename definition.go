package glslfilter

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
	"gopkg.in/yaml.v2"
)

type TextureFilterType int32

type TextureDefinition struct {
	Path   string
	Name   string
	Filter TextureFilterType
}

type Definition struct {
	Render struct {
		Width  int
		Height int
	}
	Stages []struct {
		FragmentShaderPath string `yaml:"fragmentShaderPath"`
		Textures           []TextureDefinition
	}
}

func LoadDefinitionFromStdin() (definition Definition, err error) {
	definitionString, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return definition, err
	}

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
