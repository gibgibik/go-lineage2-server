package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	text := `
 asd
d  g
g
`
	reg := regexp.MustCompile("\\s+")
	fmt.Println(reg.ReplaceAllString(strings.ReplaceAll(text, "\n", " "), " "))
}
