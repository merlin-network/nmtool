package main

import (
	"github.com/fanfury-sports/nmtool/contrib/update-genesis-validators/cmd"
	"github.com/merlin-network/nemo/app"
)

func main() {
	app.SetSDKConfig()
	cmd.Execute()
}
