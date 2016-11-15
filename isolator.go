package main

import (
	"github.com/root-gg/isolator/server"
)

func main() {
	izolator := server.NewIzolatorServer()
	izolator.Start()
}
