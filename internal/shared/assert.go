package shared

import "log"

func Assert(test bool, message string) {
	if !test {
		log.Panic(message)
	}
}
