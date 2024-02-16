package main

import "strings"

func getArgument(input string, index int) (string, bool) {
	args := strings.Split(input, " ")
	if index >= len(args) || index < 0 {
		return "", false
	}
	return args[index], true
}
