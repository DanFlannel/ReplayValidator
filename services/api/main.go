package main

import (
	libcmd "github.com/sid-technologies/replayvalidator/pkg/cmd"
	appcmd "github.com/sid-technologies/replayvalidator/services/api/cmd"
)

func main() {
	libcmd.Main(appcmd.New())
}
