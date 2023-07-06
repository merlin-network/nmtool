package main

import (
	"github.com/merlin-network/nmtool/contrib/update-genesis-validators/cmd"
	"github.com/merlin-network/nemo/app"
)

func main() {
	app.SetSDKConfig()
	cmd.Execute()
}
