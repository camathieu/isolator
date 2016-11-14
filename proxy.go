package main

import (
	"github.com/root-gg/isolator/proxy"
)

func main() {
	// Load configuration
	config := proxy.NewProxyConfig()
	config.Name = "localhost"
	config.Targets = append(config.Targets,"ws://localhost:8080/proxy")

	i := proxy.NewProxy(config)
	i.Start()

	select {}
}

