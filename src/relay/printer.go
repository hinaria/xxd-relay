package relay

import "fmt"

func printf(format string, items ...interface{}) {
	fmt.Printf(format, items...)
}

func println(items ...interface{}) {
	fmt.Println(items...)
}
