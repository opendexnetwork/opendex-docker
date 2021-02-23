package main

import (
	"github.com/opendexnetwork/opendex-docker/launcher/cmd"
	"os"
)


func main() {
	if err := os.Setenv("DOCKER_API_VERSION", "1.40"); err != nil {
		panic(err)
	}
	cmd.Execute()
}
