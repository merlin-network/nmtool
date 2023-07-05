package main

import (
	"github.com/incubus-network/nemo/app"
	"github.com/fanfury-sports/nmtool/contrib/update-genesis-validators/cmd"
)

func main() {
	app.SetSDKConfig()
	cmd.Execute()
}
