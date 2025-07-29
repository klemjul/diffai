package main

import (
	"fmt"
	"os"

	"github.com/klemjul/diffai/cmd"
	"github.com/klemjul/diffai/internal/app"
)

func fatalOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	services := app.NewDefaultServiceProvider()
	rootCmt, err := cmd.RootCommand(services)
	fatalOnError(err)
	rootCmt.Execute()
}
