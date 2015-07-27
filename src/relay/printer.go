package relay

import (
	"fmt"
	"time"
)

var (
	layout      = "2006-01-02 03:04:05pm"
	location, _ = time.LoadLocation("Australia/Melbourne")
)

func timeString() string {
	t := time.Now()

	if location != nil {
		t = t.In(location)
	}

	return "[" + t.Format(layout) + "]"
}

func printf(format string, items ...interface{}) {
	fmt.Printf(timeString() + " " + format, items...)
}

func println(items ...interface{}) {
	params := make([]interface{}, 0, len(items)+1)
	params = append(params, timeString())
	params = append(params, items...)

	fmt.Println(params...)
}
