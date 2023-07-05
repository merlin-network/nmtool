package main

import (
	"github.com/fanfury-sports/nmtool/contrib/update-genesis-validators/cmd"
	"github.com/incubus-network/nemo/app"
)

func main() {
	app.SetSDKConfig()
	cmd.Execute()
}
