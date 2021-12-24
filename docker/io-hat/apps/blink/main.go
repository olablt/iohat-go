package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	io "github.com/olablt/iohat"
)

func main() {

	mqtt_host := os.Getenv("MQTT_HOST")

	// BOARD0: RPI
	rpi := io.NewRPI()
	// SEND
	go func() {
		ticker_publish_all := time.NewTicker(time.Second * 1)
		count := 0
		for {
			count++
			select {
			case <-ticker_publish_all.C:
				// log.Println("sending out1")
				if count%2 == 0 {
					rpi.Set("out1", true)
					rpi.Set("out2", !true)
					rpi.Set("out3", true)
					rpi.Set("out4", !true)
					rpi.Set("out5", true)
					rpi.Set("out6", !true)
				} else {
					rpi.Set("out1", !true)
					rpi.Set("out2", true)
					rpi.Set("out3", !true)
					rpi.Set("out4", true)
					rpi.Set("out5", !true)
					rpi.Set("out6", true)
				}
				// log.Println("ending sending setch")
			}
		}
	}()
	b := io.NewBoard("rpi1", rpi, nil)
	b.ConnectMQTT(mqtt_host)
	b.Monitor()
	log.Println(rpi)

	// BOARD1: extension
	ext := io.NewExt()
	// SEND
	go func() {
		ticker_publish_all := time.NewTicker(time.Second * 1)
		count := 0
		for {
			count++
			select {
			case <-ticker_publish_all.C:
				if count%2 == 0 {
					ext.Set("out1", !true)
					ext.Set("out2", true)
					ext.Set("out3", !true)
					ext.Set("out4", true)
					ext.Set("out5", !true)
					ext.Set("out6", true)
				} else {
					ext.Set("out1", true)
					ext.Set("out2", !true)
					ext.Set("out3", true)
					ext.Set("out4", !true)
					ext.Set("out5", true)
					ext.Set("out6", !true)
				}
				log.Println("ending sending setch")
			}
		}
	}()
	eb := io.NewBoard("ext1", ext, nil)
	eb.ConnectMQTT(mqtt_host)
	eb.Monitor()

	// Messages will be delivered asynchronously so we just need to wait for a signal to shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	fmt.Println("signal caught - exiting")
	// b.Close()
	// c.Disconnect(250)
	fmt.Println("shutdown complete")
}
