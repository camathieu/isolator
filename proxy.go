package main

import (
	"github.com/root-gg/isolator/proxy"
	"os"
	"os/signal"
	"log"
	"github.com/nu7hatch/gouuid"
)

func main() {
	// Load configuration
	config := proxy.NewProxyConfig()

	//config.Name = "localhost"
	hostname, _ := uuid.NewV4()
	config.Name = hostname.String()

	config.Targets = append(config.Targets,"ws://localhost:8080/register")

	proxy := proxy.NewProxy(config)

	/*
 	* Handle SIGINT
 	*/
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for {
			<-c
			log.Println("SIGINT Detected")
			proxy.Shutdown()
			os.Exit(0)
		}
	}()

	proxy.Start()

	select {}
}

