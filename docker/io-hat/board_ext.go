package iohat

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/d2r2/go-i2c"
	"github.com/d2r2/go-logger"
)

func NewExt() *ext {
	outputs := make(map[string]int)
	outputs["out1"] = 13
	outputs["out2"] = 12
	outputs["out3"] = 11
	outputs["out4"] = 10
	outputs["out5"] = 9
	outputs["out6"] = 8
	// outputs["out7"] = 1

	inputs := make(map[string]int)
	inputs["in1"] = 0
	inputs["in2"] = 1
	inputs["in3"] = 2
	inputs["in4"] = 3
	inputs["in5"] = 4
	inputs["in6"] = 5
	inputs["in7"] = 6
	inputs["in8"] = 7
	inputs["in9"] = 15
	inputs["in10"] = 14

	lastDebounceTime := make(map[string]time.Time)

	// prepare values
	values := make(map[int]bool)
	for _, k := range outputs {
		values[k] = true
	}
	for _, k := range inputs {
		values[k] = true
	}
	lastValues := make(map[int]bool)
	for _, k := range inputs {
		lastValues[k] = true
	}
	// log.Println("values", values)

	r := &ext{
		inputs:           inputs,
		outputs:          outputs,
		values:           values,
		lastValues:       lastValues,
		lastDebounceTime: lastDebounceTime,
		setCh:            make(chan *PortValChange),
	}

	return r
}

func (b *ext) Monitor() chan *PortValChange {

	getCh := make(chan *PortValChange)
	// setCh := make(chan *PortValChange)

	go func() {
		for {
			log.Println("connecting....")
			err := b.Connect()
			if err != nil {
				fmt.Println("error:", err)
			} else {
				log.Println("connected")
				ticker_monitor := time.NewTicker(time.Millisecond * 10)
			loop:
				for {
					select {
					case in := <-b.setCh:
						// log.Println("got input", in)
						if in.command == "toggle" {
							// toggle
							b.toggle(in.port)
						} else {
							b.set(in.port, in.value)
						}
						// b.set(in.port, in.value)

					case <-ticker_monitor.C:
						// check for new values
						newValues, err := b.getNewValues()
						if err != nil {
							break loop
						}
						// log.Println(newValues)
						if len(newValues) > 0 {
							// inputs
							for port, j := range b.inputs {
								// debouncing source https://www.arduino.cc/en/Tutorial/BuiltInExamples/Debounce
								// if last check and last value are different - store time
								if b.lastValues[j] != newValues[j] {
									// log.Println("[debouncing] got new input")
									b.lastDebounceTime[port] = time.Now()
								}
								now := time.Now()
								// if last difference is older than 50ms
								if now.Sub(b.lastDebounceTime[port]).Milliseconds() > 50 {
									// if last official value is not the same as las new  val
									if b.values[j] != newValues[j] {
										// inform channel and store new val
										getCh <- &PortValChange{port, newValues[j], "get"}
										b.values[j] = newValues[j]
									}
								}
								// store last value for next round check
								b.lastValues[j] = newValues[j]
							}
							// outputs
							for port, j := range b.outputs {
								if b.values[j] != newValues[j] {
									getCh <- &PortValChange{port, newValues[j], "get"}
									b.values[j] = newValues[j]
								}
							}
						}
					}
				}
				b.Close()
				if err != nil {
					fmt.Println("error disconnecting:", err)
				}

			}

			// sleep before reconnect
			time.Sleep(time.Millisecond * 500)
		}
		// log.Println("ups")

	}()
	return getCh
}

// get word bytes from i2c
func (r *ext) getW() (uint16, error) {
	// buff := make([]byte, 0, 1)
	// buff := []byte{0xFF, 0xFF}
	// buff := []byte{0x00, 0x00}
	buff := make([]byte, 2)
	_, err := r.i2c.ReadBytes(buff)
	w := binary.BigEndian.Uint16(buff)
	if err != nil {
		return 0, err
	}
	return w, err
}

func (r *ext) getNewValues() (map[int]bool, error) {
	w, err := r.getW()
	if err != nil {
		return nil, err
	}
	// b := make([]bool, 16)
	b := make(map[int]bool)
	for i := 0; i < 16; i++ {
		if w&(1<<i) == 0 {
			b[15-i] = false
			// log.Println("False")
		} else {
			b[15-i] = true
			// log.Println("True")
		}
		// log.Println(v.Get(i))
	}
	// log.Println(b)
	return b, nil
}

func (r *ext) set(port string, value bool) error {
	// if _, ok := r.outputs[port]; ok {
	// 	r.pcf.Set(r.outputs[port], value)
	// }
	out_no := r.outputs[port]
	if out_no > 15 || out_no < 0 {
		log.Println("output value is not in interval 0..15")
		return nil
	}
	w, err := r.getW()
	if err != nil {
		return err
	}
	if value {
		// set bit
		w = w | (1 << (15 - out_no))
	} else {
		// unset bit
		w = w &^ (1 << (15 - out_no))
	}
	for _, k := range r.inputs {
		// log.Println("i, k", i, k)
		// set input bits to 1
		w = w | (1 << (15 - k))
	}
	// fmt.Printf("after unset: %08b\n", w)
	buff := make([]byte, 2)
	binary.BigEndian.PutUint16(buff, w)
	// binary.LittleEndian.PutUint16(buff, w)
	_, err = r.i2c.WriteBytes(buff)
	return err
}
func (r *ext) toggle(port string) error {
	out_no := r.outputs[port]
	if out_no > 15 || out_no < 0 {
		log.Println("output value is not in interval 0..15")
		return nil
	}
	w, err := r.getW()
	if err != nil {
		return err
	}
	w = w ^ (1 << (15 - out_no))
	for _, k := range r.inputs {
		// log.Println("i, k", i, k)
		// set input bits to 1
		w = w | (1 << (15 - k))
	}
	// fmt.Printf("after unset: %08b\n", w)
	buff := make([]byte, 2)
	binary.BigEndian.PutUint16(buff, w)
	// binary.LittleEndian.PutUint16(buff, w)
	_, err = r.i2c.WriteBytes(buff)
	return err
}
func (r *ext) Set(port string, value bool) error {
	r.setCh <- &PortValChange{port, value, "set"}
	return nil
}
func (r *ext) Toggle(port string) error {
	r.setCh <- &PortValChange{port, false, "toggle"}
	return nil
}

func (r *ext) Close() {
	r.i2c.Close()
	// return nil
}
func (r *ext) Connect() error {

	var err error
	// Create new connection to I2C bus on 2 line with address 0x27
	r.i2c, err = i2c.NewI2C(0x20, 1)
	logger.ChangePackageLogLevel("i2c", logger.InfoLevel)
	if err != nil {
		return err
	}

	err = r.Reset()
	if err != nil {
		return err
	}
	return nil
}

// func (r *ext) Get(port string) bool {
// 	value, _ := r.values[r.outputs[port]]
// 	return value
// 	// value, _ := r.pcf.Get(r.outputs[port])
// 	// return value
// }

func (r *ext) Reset() error {
	_, err := r.i2c.WriteBytes([]byte{0xFF, 0xFF})
	// r.pcf.Set(r.outputs[port], value)
	return err
}

func (r *ext) GetInputs() map[string]int {
	return r.inputs
}
func (r *ext) GetOutputs() map[string]int {
	return r.outputs
}
func (r *ext) GetValues() map[int]bool {
	// make copy of values
	values := make(map[int]bool)
	for i, val := range r.values {
		values[i] = val
	}
	return values
}
