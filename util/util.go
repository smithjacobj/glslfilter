package util

import (
	"log"
)

func Invariant(i interface{}, extras ...interface{}) {
	switch v := i.(type) {
	case error:
		if v != nil {
			log.Fatalln(v)
		}
	case bool:
		if !v {
			if len(extras) > 0 {
				log.Fatalln(extras...)
			}
			log.Fatalln("boolean invariant failure")
		}
	}
}
