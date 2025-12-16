package container

import (
	"os"

	"github.com/opencontainers/runc/libcontainer"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

func Init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		libcontainer.Init()

		os.Exit(0)
	}
}
