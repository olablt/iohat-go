package iohat

import (
	"fmt"
	"log"
	"os"
	"time"

	// rpio "github.com/stianeikeland/go-rpio"
	rpio "github.com/stianeikeland/go-rpio/v4"
)

func NewRPI() *rpi {
	outputs := make(map[string]int)
	outputs["out1"] = 11
	outputs["out2"] = 5
	outputs["out3"] = 6
	outputs["out4"] = 13
	outputs["out5"] = 19
	outputs["out6"] = 26
	outputs["out7"] = 8

	inputs := make(map[string]int)
	inputs["in1"] = 18
	inputs["in2"] = 23
	inputs["in3"] = 4
	inputs["in4"] = 17
	inputs["in5"] = 27
	inputs["in6"] = 22
	inputs["in7"] = 10
	inputs["in8"] = 9
	inputs["in9"] = 24
	inputs["in10"] = 25

	// prepare values
	values := make(map[int]bool)
	for _, k := range outputs {
		values[k] = true
	}
	for _, k := range inputs {
		values[k] = true
	}

	r := &rpi{
		inputs:  inputs,
		outputs: outputs,
		values:  values,
		setCh:   make(chan *PortValChange),
	}
	r.Init()
	// go r.Monitor()
	return r
}

func (r *rpi) Monitor() chan *PortValChange {

	getCh := make(chan *PortValChange)

	go func() {
		for {
			log.Println("connecting....")
			ticker_monitor := time.NewTicker(time.Millisecond * 100)
			for {
				// count++
				select {
				case in := <-r.setCh:
					// log.Println("got input", in)
					if in.command == "toggle" {
						// toggle
						r.toggle(in.port)
					} else {
						r.set(in.port, in.value)
					}

				case <-ticker_monitor.C:
					// log.Println("monitoring")
					// log.Println("starting new val block")
					// getCh <- &PortValChange{"in2", true}
					// log.Println("ending new val block")
					// INPUT
					newValues, _ := r.getNewInputValues()
					// if err != nil {
					// 	break loop
					// }
					// log.Println("input", newValues)
					if len(newValues) > 0 {
						for port, j := range r.inputs {
							if r.values[j] != newValues[j] {
								getCh <- &PortValChange{port, newValues[j], "get"}
								r.values[j] = newValues[j]
							}
						}
					}
					// OUTPUT
					newValues, _ = r.getNewOutputValues()
					// log.Println("output", newValues)
					// if err != nil {
					// 	break loop
					// }
					if len(newValues) > 0 {
						for port, j := range r.outputs {
							if r.values[j] != newValues[j] {
								getCh <- &PortValChange{port, newValues[j], "get"}
								r.values[j] = newValues[j]
							}
						}
					}
				}
			}
			// err = b.conn.Close()
			// if err != nil {
			// 	fmt.Println("error disconnecting:", err)
			// }

			// sleep before reconnect
			time.Sleep(time.Millisecond * 500)
		}

	}()
	return getCh

}

func (r *rpi) Set(port string, value bool) error {
	r.setCh <- &PortValChange{port, value, "set"}
	return nil
}
func (r *rpi) Toggle(port string) error {
	r.setCh <- &PortValChange{port, false, "toggle"}
	return nil
}
func (r *rpi) set(port string, value bool) {
	// r.pcf.Set(r.outputs[port], value)
	if _, ok := r.outputs[port]; ok {
		if value {
			rpio.WritePin(rpio.Pin(r.outputs[port]), rpio.High)
		} else {
			rpio.WritePin(rpio.Pin(r.outputs[port]), rpio.Low)
		}
	}
	// return nil
}
func (r *rpi) toggle(port string) {
	// r.pcf.Set(r.outputs[port], value)
	if _, ok := r.outputs[port]; ok {
		rpio.TogglePin(rpio.Pin(r.outputs[port]))
	}
	// return nil
}

// func (r *rpi) Get(port string) bool {
// 	value, _ := r.values[r.outputs[port]]
// 	return value
// 	// return rpio.ReadPin(rpio.Pin(r.outputs[port])) == rpio.High
// }

func (r *rpi) getNewInputValues() (map[int]bool, error) {
	newValues := make(map[int]bool)
	for _, bcm := range r.inputs {
		newValues[bcm] = rpio.ReadPin(rpio.Pin(bcm)) == rpio.High
	}
	return newValues, nil
}

func (r *rpi) getNewOutputValues() (map[int]bool, error) {
	newValues := make(map[int]bool)
	for _, bcm := range r.outputs {
		newValues[bcm] = rpio.ReadPin(rpio.Pin(bcm)) == rpio.High
	}
	return newValues, nil
}

// func (r *rpi) updateValues() error {
// 	// log.Println("values", b)
// 	return nil
// }

func (r *rpi) Close() {
	// r.pcf.Set(r.outputs[port], value)
	rpio.Close()
}

func (r *rpi) Init() {
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Unmap gpio memory when done

	for _, bcm := range r.outputs {
		// rpio.PullMode(rpio.Pin(bcm), rpio.PullUp)
		rpio.PinMode(rpio.Pin(bcm), rpio.Output)
		rpio.WritePin(rpio.Pin(bcm), rpio.High)
		// log.Println("setting output for: ", port, bcm)
	}
	for _, bcm := range r.inputs {
		rpio.PinMode(rpio.Pin(bcm), rpio.Input)
		rpio.PullMode(rpio.Pin(bcm), rpio.PullDown)
		// rpio.PullMode(rpio.Pin(bcm), rpio.PullOff)
		rpio.WritePin(rpio.Pin(bcm), rpio.High)
		// log.Println("setting input for: ", port, bcm)
	}

	log.Println("initialized")
}

func (r *rpi) GetInputs() map[string]int {
	return r.inputs
}
func (r *rpi) GetOutputs() map[string]int {
	return r.outputs
}
func (r *rpi) GetValues() map[int]bool {
	// make copy of values
	values := make(map[int]bool)
	for i, val := range r.values {
		values[i] = val
	}
	return values
}
