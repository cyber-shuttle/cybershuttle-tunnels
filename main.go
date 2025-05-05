package main

import (
	"fmt"
	"os"

	"github.com/cyber-shuttle/cybershuttle-tunnels/client"
	"github.com/cyber-shuttle/cybershuttle-tunnels/server"

	//"os/signal"
	"strconv"
	//"syscall"

	"github.com/fatedier/frp/pkg/util/log"
)

func main() {
	fmt.Println("Starting…")
	compName := os.Args[1]
	cfgFilePath := os.Args[2]

	if compName == "client" {
		log.Infof(("Line before client run"))

		err, _, errChan, _ := client.RunClient(cfgFilePath)

		if err != nil {
			log.Errorf("frpc service for config file [%s] failed: %v", cfgFilePath, err)
			os.Exit(1)
		}
		if err := <-errChan; err != nil {
			log.Errorf("Error running server: %v", err)
		}
	}

	if compName == "server" {
		apiPort := os.Args[3]
		port, err := strconv.Atoi(apiPort)
		if err != nil {
			log.Errorf("Invalid API port [%s]: %v", apiPort, err)
			os.Exit(1)
		}
		log.Infof("Starting API server on port [%d]", port)
		go server.StartAPIServer(port)

		if err := server.RunServer(cfgFilePath); err != nil {
			log.Errorf("frps service for config file [%s] failed: %v", cfgFilePath, err)
			os.Exit(1)
		}

	}

	// Wait for interrupt signal

	log.Infof("frpc service for config file [%s] stopped", cfgFilePath)
	os.Exit(0)
}
