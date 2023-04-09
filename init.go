package main

import (
	"github.com/PretendoNetwork/splatoon-secure/database"
	"github.com/PretendoNetwork/splatoon-secure/globals"
	"github.com/PretendoNetwork/splatoon-secure/utility"
)

func init() {

	globals.Config, _ = utility.ImportConfigFromFile("secure.config")

	database.ConnectAll()
}
