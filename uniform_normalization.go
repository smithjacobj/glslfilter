package glslfilter

import (
	"fmt"
	"reflect"

	"github.com/smithjacobj/glslfilter/util"
)

func normalizeUniformValue(definition UniformDefinition) (normed interface{}) {
	if definition.Type.IsArray {
		return normalizeArray(definition.Value, definition.Type)
	} else {
		return normalizeValue(definition.Value, definition.Type)
	}
}

func normalizeArray(v interface{}, typ UniformType) (normed interface{}) {
	vValue := reflect.ValueOf(v)
	util.Invariant(vValue.Kind() == reflect.Slice)

	length := vValue.Len()

	switch typ.ScalarType {
	case Float:
		var floatSlice []float32
		for i := 0; i < length; i++ {
			normedAsType := normToFloat(vValue.Index(i).Interface())
			validateTypeLength(normedAsType, typ)
			floatSlice = append(floatSlice, normedAsType...)
		}
		normed = floatSlice
	case Int:
		var intSlice []int32
		for i := 0; i < length; i++ {
			normedAsType := normToInt(vValue.Index(i).Interface())
			validateTypeLength(normedAsType, typ)
			intSlice = append(intSlice, normedAsType...)
		}
		normed = intSlice
	case Uint:
		var uintSlice []uint32
		for i := 0; i < length; i++ {
			normedAsType := normToUint(vValue.Index(i).Interface())
			validateTypeLength(normedAsType, typ)
			uintSlice = append(uintSlice, normedAsType...)
		}
		normed = uintSlice
	default:
		panic("normalizeArray: invalid type specified")
	}
	return normed
}

func normalizeValue(v interface{}, typ UniformType) (normed interface{}) {
	switch typ.ScalarType {
	case Float:
		normed = normToFloat(v)
	case Int:
		normed = normToInt(v)
	case Uint:
		normed = normToUint(v)
	default:
		panic("normalizeValue: invalid type specified")
	}
	validateTypeLength(normed, typ)
	return normed
}

func normToFloat(v interface{}) (normed []float32) {
	switch v := v.(type) {
	case float32:
		return []float32{v}
	case []float32:
		return v
	case float64:
		return []float32{float32(v)}
	case []float64:
		f64Slice := v
		f32Slice := []float32{}
		for _, v := range f64Slice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	case int:
		return []float32{float32(v)}
	case []int:
		intSlice := v
		f32Slice := []float32{}
		for _, v := range intSlice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	case uint:
		return []float32{float32(v)}
	case []uint:
		uintSlice := v
		f32Slice := []float32{}
		for _, v := range uintSlice {
			f32Slice = append(f32Slice, float32(v))
		}
		return f32Slice
	case []interface{}:
		// special case from reflection
		interfaceSlice := v
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
	switch v := v.(type) {
	case float32:
		return []int32{int32(v)}
	case []float32:
		f32Slice := v
		intSlice := []int32{}
		for _, v := range f32Slice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	case float64:
		return []int32{int32(v)}
	case []float64:
		f64Slice := v
		intSlice := []int32{}
		for _, v := range f64Slice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	case int:
		return []int32{int32(v)}
	case []int:
		intSlice := v
		int32Slice := []int32{}
		for _, v := range intSlice {
			int32Slice = append(int32Slice, int32(v))
		}
		return int32Slice
	case uint:
		return []int32{int32(v)}
	case []uint:
		uintSlice := v
		intSlice := []int32{}
		for _, v := range uintSlice {
			intSlice = append(intSlice, int32(v))
		}
		return intSlice
	case []interface{}:
		// special case from reflection
		interfaceSlice := v
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
	switch v := v.(type) {
	case float32:
		return []uint32{uint32(v)}
	case []float32:
		f32Slice := v
		uintSlice := []uint32{}
		for _, v := range f32Slice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case float64:
		return []uint32{uint32(v)}
	case []float64:
		f64Slice := v
		uintSlice := []uint32{}
		for _, v := range f64Slice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case int:
		return []uint32{uint32(v)}
	case []int:
		intSlice := v
		uintSlice := []uint32{}
		for _, v := range intSlice {
			uintSlice = append(uintSlice, uint32(v))
		}
		return uintSlice
	case uint:
		return []uint32{uint32(v)}
	case []uint:
		uintSlice := v
		uint32Slice := []uint32{}
		for _, v := range uintSlice {
			uint32Slice = append(uint32Slice, uint32(v))
		}
		return uint32Slice
	case []interface{}:
		// special case from reflection
		interfaceSlice := v
		uint32Slice := []uint32{}
		for _, v := range interfaceSlice {
			uint32Slice = append(uint32Slice, normToUint(v)...)
		}
		return uint32Slice
	}
	panicMsg := fmt.Sprintf("normToUint: unexpected type %s in bagging area", reflect.TypeOf(v))
	panic(panicMsg)
}

func validateTypeLength(v interface{}, typ UniformType) {
	vValue := reflect.ValueOf(v)
	correctedSize := typ.VectorSize
	if correctedSize == 0 {
		// scalars are 0, but will have a 1-entry slice due to normalization
		correctedSize = 1
	}
	util.Invariant(vValue.Kind() == reflect.Slice)
	util.Invariant(vValue.Len() == correctedSize)
}
