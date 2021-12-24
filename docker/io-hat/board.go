package iohat

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/d2r2/go-i2c"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type rpi struct {
	BoardID string // ext1
	outputs map[string]int
	inputs  map[string]int
	values  map[int]bool // output values
	setCh   chan *PortValChange
	mutex   sync.Mutex
}

type ext struct {
	BoardID          string // ext1
	outputs          map[string]int
	inputs           map[string]int
	values           map[int]bool // output values
	lastValues       map[int]bool // output values
	lastDebounceTime map[string]time.Time
	// pcf     *PCF8575
	setCh chan *PortValChange
	mutex sync.Mutex
	i2c   *i2c.I2C
}

type ds3484 struct {
	BoardID string // ext1
	outputs map[string]int
	inputs  map[string]int
	values  map[int]bool // output values
	ip      string
	conn    net.Conn
	setCh   chan *PortValChange
	// connected bool
	// pcf     *PCF8575
	mutex sync.Mutex
}

type PortAlias struct {
	// PortName   string
	Topic   string
	TurnOFF int // turn off after given period ms
}

type board interface {
	Set(string, bool) error
	Toggle(string) error
	// Get(string) bool
	// Reset()
	// Get(string) bool
	GetValues() map[int]bool
	GetOutputs() map[string]int
	GetInputs() map[string]int
	Monitor() chan *PortValChange
	// Monitor() (chan *PortValChange, chan *PortValChange)
	// getCh() chan *PortValChange
	// setCh() chan *PortValChange
}

type Board struct {
	MQTTClient mqtt.Client
	BoardID    string // ext1
	r          board
	aliases    map[string]PortAlias
}

type PortValChange struct {
	port    string
	value   bool
	command string
}

func NewBoard(BoardID string, brd board, aliases map[string]PortAlias) *Board {

	b := &Board{
		BoardID: BoardID,
		// values:  make(map[int]bool),
		aliases: aliases,
		r:       brd,
	}

	return b
}

// monitor inputs and periodical publish status
func (b *Board) Monitor() {

	getCh := b.r.Monitor()
	go func() {
		// ticker_monitor := time.NewTicker(time.Millisecond * 500)
		ticker_publish_all := time.NewTicker(time.Second * 30)
		// defer ticker_monitor.Stop()
		defer ticker_publish_all.Stop()

		inputs := b.r.GetInputs()
		outputs := b.r.GetOutputs()

		values := b.r.GetValues()
		for port, i := range inputs {
			b.publishState(port, values[i], false, false)
			// b.publishState(port, values[i], true)
		}
		for port, i := range outputs {
			b.publishState(port, values[i], false, false)
			// b.publishState(port, values[i], true)
		}

		// count := 1
		for {
			// count++
			select {
			case newVal := <-getCh:
				b.publishState(newVal.port, newVal.value, true, true)
			case <-ticker_publish_all.C:
				// publish all stored values
				values := b.r.GetValues()
				for port, i := range inputs {
					b.publishState(port, values[i], false, false)
					// b.publishState(port, values[i], true)
				}
				for port, i := range outputs {
					b.publishState(port, values[i], false, false)
					// b.publishState(port, values[i], true)
				}
			}
		}
	}()
}

// publish port value
func (b *Board) publishState(port string, state, isNewVal, log bool) {
	// set value by state
	value := "0"
	if state == false {
		value = "1"
	}

	// set topic and value
	var topic string
	if alias, ok := b.aliases[port]; ok {
		topic = fmt.Sprintf("stat/%s", alias.Topic)
	} else {
		topic = fmt.Sprintf("stat/%s/%v", b.BoardID, port)
	}

	// log
	// log = true
	if log {
		b.Log("publishing: topic:", topic, " val:", value)
	}
	b.publish(topic, value)

	if isNewVal {
		if state == false {
			topic = fmt.Sprintf("stat/%s/%v/on", b.BoardID, port)
			b.publish(topic, value)
			if log {
				b.Log("publishing: topic:", topic, " val:", value)
			}
		}
	}
}

// mqtt publish
func (b *Board) publish(topic, msg string) {
	QOS := byte(0)
	t := b.MQTTClient.Publish(topic, QOS, false, msg)
	// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
	go func() {
		_ = t.Wait() // Can also use '<-t.Done()' in releases > 1.2.0
		if t.Error() != nil {
			fmt.Printf("ERROR PUBLISHING: %s\n", t.Error())
		}
	}()
}

func (b *Board) ConnectMQTT(uri string) mqtt.Client {
	opts := b.subscribeMQTT(uri)

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	// mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	b.MQTTClient = mqtt.NewClient(opts)
	token := b.MQTTClient.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return b.MQTTClient

}

// log helper
func (b *Board) Log(v ...interface{}) {
	out := fmt.Sprintf("[%s] %s", b.BoardID, fmt.Sprint(v...))
	// log.Print(out)
	log.Println(out)
}

func (b *Board) Close() {
	b.Log("Closing connections")
	b.MQTTClient.Disconnect(250)
}

// prepare mqtt and subscribe cmd outputs
func (b *Board) subscribeMQTT(uri string) *mqtt.ClientOptions {
	opts := b.createMQTToptions(uri)

	opts.OnConnect = func(c mqtt.Client) {
		b.Log("MQTT connection established")

		// Establish the subscriptions - doing this here means that it will happen every time a connection is established

		// subscribe to outputs aliases
		outputs := b.r.GetOutputs()
		for port := range outputs {
			if alias, ok := b.aliases[port]; ok {
				// log.Println("subscribing to", alias.Topic, port, alias.TurnOFF)
				topic := fmt.Sprintf("cmd/%s", alias.Topic)

				t := c.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
					// only when on off
					if string(msg.Payload()) == "0" || string(msg.Payload()) == "1" || string(msg.Payload()) == "" {

						// set boolean value from on/off string
						value := false
						if string(msg.Payload()) == "0" {
							value = true // off
						}

						// get the port number
						var port string
						for p, a := range b.aliases {
							if a.Topic == alias.Topic {
								port = p
							}
						}

						if len(port) > 0 {
							b.Log("Got ", msg.Topic(), ":", string(msg.Payload()))

							// set board output
							// h.pcf.Set(h.outputs[port], value)
							b.r.Set(port, value)
							// fmt.Printf("* [%s] %s, port:%s\n", msg.Topic(), string(msg.Payload()), port)

							// if port is temporary - turn off after defined time period
							if !value && alias.TurnOFF > 0 {
								go func(alias PortAlias, port string) {
									time.Sleep(time.Duration(alias.TurnOFF) * time.Millisecond)
									b.r.Set(port, true)
									b.Log("auto off: ", alias, " ", port)
								}(alias, port)
							}

						}

					}
				})
				go func() {
					_ = t.Wait()
					if t.Error() != nil {
						out := fmt.Sprintf("MQTT ERROR SUBSCRIBING: %s\n", t.Error())
						b.Log(out)
					} else {
						out := fmt.Sprintf("MQTT subscribed to: %s", topic)
						b.Log(out)
					}
				}()
			}

		}

		// subscribe to board IO
		topic := fmt.Sprintf("cmd/%v/#", b.BoardID)
		t := c.Subscribe(topic, 0, b.handleCMD)
		// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
		// in other handlers does cause problems its best to just assume we should not block
		go func() {
			_ = t.Wait()
			if t.Error() != nil {
				out := fmt.Sprintf("MQTT ERROR SUBSCRIBING: %s\n", t.Error())
				b.Log(out)
			} else {
				out := fmt.Sprintf("MQTT subscribed to: %s", topic)
				b.Log(out)
			}
		}()
	}
	return opts
}

// handle is called when a message is received
func (b *Board) handleCMD(_ mqtt.Client, msg mqtt.Message) {
	b.Log("Got ", msg.Topic(), ":", string(msg.Payload()))

	parts := strings.Split(msg.Topic(), "/")
	payload := fmt.Sprintf("%s", msg.Payload())
	// id, _ := strconv.Atoi(stripLetters(parts[2]))

	// SET OUTPUT
	if len(parts[2]) >= 4 {
		port := parts[2]
		// set_on, set_off, set payload:on or off
		cmd := parts[3]
		// b.Log("command:", cmd)
		value := false
		if cmd == "set_off" {
			value = true
		} else if cmd == "set" {
			if payload == "0" {
				value = true
			}
		}

		outputs := b.r.GetOutputs()
		if _, ok := outputs[port]; ok {
			if cmd == "toggle" {
				b.r.Toggle(port)
			} else {
				b.r.Set(port, value)
			}
			//do something here
		} else {
			b.Log(fmt.Sprintf("ERROR unknown port: %v", port))
		}
	}
}

func (b *Board) createMQTToptions(uri string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(uri)
	opts.SetClientID(b.BoardID)

	opts.SetOrderMatters(false)       // Allow out of order messages (use this option unless in order delivery is essential)
	opts.ConnectTimeout = time.Second // Minimal delays on connect
	opts.WriteTimeout = time.Second   // Minimal delays on writes
	opts.KeepAlive = 10               // Keepalive every 10 seconds so we quickly detect network outages
	opts.PingTimeout = time.Second    // local broker so response should be quick

	// Automate connection management (will keep trying to connect and will reconnect if network drops)
	opts.ConnectRetry = true
	opts.AutoReconnect = true

	// If using QOS2 and CleanSession = FALSE then it is possible that we will receive messages on topics that we
	// have not subscribed to here (if they were previously subscribed to they are part of the session and survive
	// disconnect/reconnect). Adding a DefaultPublishHandler lets us detect this.
	opts.DefaultPublishHandler = func(_ mqtt.Client, msg mqtt.Message) {
		out := fmt.Sprintf("UNEXPECTED MESSAGE: %s\n", msg)
		b.Log(out)
	}

	// Log events
	opts.OnConnectionLost = func(cl mqtt.Client, err error) {
		b.Log("connection lost")
	}

	opts.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		b.Log("attempting to reconnect")
	}
	return opts
}
