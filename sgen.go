package main

import (
	"os"

	"github.com/scnewma/sgen/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
