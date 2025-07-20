package main

import (
	"github.com/klemjul/diffai/cmd"
	"github.com/klemjul/diffai/internal/app"
)

func main() {
	app := app.NewDefaultApp()
	cmd.RootCommand(app).Execute()
}
