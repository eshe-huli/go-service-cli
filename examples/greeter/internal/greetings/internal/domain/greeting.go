package domain

import "strings"

func Greeting(name string) string { return "Hello, " + strings.TrimSpace(name) + "!" }
