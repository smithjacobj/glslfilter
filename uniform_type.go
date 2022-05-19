package glslfilter

import (
	"fmt"
	"strings"
)

type ScalarTypeName uint

const (
	_ ScalarTypeName = iota
	Float
	Int
	Uint
)

type UniformType struct {
	ScalarType ScalarTypeName
	VectorSize int
	IsArray    bool
}

func (uniformType *UniformType) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	const kArrayPrefix = "[]"
	const kFloatPrefix = "float"
	const kIntPrefix = "int"
	const kUintPrefix = "uint"
	const kVecPrefix = "vec"

	var rawString string
	if err = unmarshal(&rawString); err != nil {
		return err
	}
	rawString = strings.ToLower(rawString)

	if strings.HasPrefix(rawString, kArrayPrefix) {
		uniformType.IsArray = true
		rawString = rawString[len(kArrayPrefix):]
	}

	if strings.HasPrefix(rawString, kFloatPrefix) {
		uniformType.ScalarType = Float
		rawString = rawString[len(kFloatPrefix):]
	} else if strings.HasPrefix(rawString, kIntPrefix) {
		uniformType.ScalarType = Int
		rawString = rawString[len(kIntPrefix):]
	} else if strings.HasPrefix(rawString, kUintPrefix) {
		uniformType.ScalarType = Uint
		rawString = rawString[len(kUintPrefix):]
	} else {
		return fmt.Errorf("invalid scalar type specified: \"%s\", options are (float|int|uint)", rawString)
	}

	if strings.HasPrefix(rawString, kVecPrefix) {
		rawString = rawString[len(kVecPrefix):]
		switch rawString {
		case "2":
			uniformType.VectorSize = 2
		case "3":
			uniformType.VectorSize = 3
		case "4":
			uniformType.VectorSize = 4
		default:
			return fmt.Errorf("invalid vector size specified, options are 2,3,4")
		}
	}

	return nil
}
