package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const configPathEnv = "DDNS_CONFIG_PATH"
const defaultTTL = 300

var (
	stopSignalChan = make(chan os.Signal, 1)
	configPath     string
)

type GracefulExit struct {
	stop chan bool
	wg   sync.WaitGroup
}

func NewGracefulExit() *GracefulExit {
	exit := &GracefulExit{
		stop: make(chan bool),
	}
	signal.Notify(stopSignalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignalChan
		fmt.Println("🛑 Stopping main thread...")
		close(exit.stop)
	}()
	return exit
}

func main() {

	ddns := NewCfDDns().LoadConfig()
	gracefulExit := NewGracefulExit()

	if len(os.Args) > 1 && os.Args[1] == "--repeat" {
		ticker := time.NewTicker(time.Duration(ddns.config.Ttl) * time.Second)
		for {
			select {
			case <-ticker.C:
				ddns.Run()
			case <-gracefulExit.stop:
				ticker.Stop()
				fmt.Println("Stopped Cloudflare DDNS updater.")
				return
			}
		}
	} else {
		ddns.Run()
	}
}
