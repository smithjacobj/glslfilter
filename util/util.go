package util

import (
	"log"
	"runtime"
)

func Invariant(i interface{}, extras ...interface{}) {
	stack := make([]byte, 1024)
	runtime.Stack(stack, false)

	switch v := i.(type) {
	case error:
		if v != nil {
			log.Fatalf("%s\n%s\n", v, stack)
		}
	case bool:
		if !v {

			if len(extras) > 0 {
				log.Fatalln(extras...)
			}
			log.Fatalf("boolean invariant failure\n%s\n", stack)
		}
	}
}
