package utils

import (
	"fmt"
	"strings"
)

type Answer string

const (
	YES Answer = "yes"
	NO Answer = "no"
)

func YesNo(question string, defaultAnswer Answer) Answer {
	var reply string
	for {
		switch defaultAnswer {
		case YES:
			fmt.Println(question + " [Y/n] ")
		case NO:
			fmt.Println(question + " [y/N] ")
		}
		_, err := fmt.Scanln(&reply)
		if err != nil {
			panic(err)
		}
		reply = strings.TrimSpace(reply)
		reply = strings.ToLower(reply)
		if reply == "y" || reply == "yes" {
			return YES
		} else if reply == "n" || reply == "no" {
			return NO
		} else if reply == "" {
			return defaultAnswer
		}
	}
}
