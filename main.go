package main

import (
	"spm/cmd"
)

func main() {
	cmd.Version = VERSION
	cmd.Execute()
}
