package main

import (
	"github.com/klemjul/diffai/cmd"
	"github.com/klemjul/diffai/internal/app"
)

func main() {
	services := app.NewDefaultServiceProvider()
	cmd.RootCommand(services).Execute()
}
