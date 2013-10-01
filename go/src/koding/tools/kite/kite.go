package kite

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"koding/db/mongodb/modelhelper"
	"koding/tools/amqputil"
	"koding/tools/dnode"
	"koding/tools/lifecycle"
	"koding/tools/log"
	"koding/virt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Kite struct {
	Name              string
	Handlers          map[string]Handler
	ServiceUniqueName string
	LoadBalancer      func(correlationName string, username string, deadService string) string
}

type Handler struct {
	Concurrent bool
	Callback   func(args *dnode.Partial, channel *Channel) (interface{}, error)
}

type Channel struct {
	Username        string
	RoutingKey      string
	CorrelationName string
	Alive           bool
	KiteData        interface{}
	onDisconnect    []func()
}

// Control is used by controlChannel to shutdown shutdown VM's associated with
// their hostnameAlias.
type Control struct {
	HostnameAlias string
}

func New(name string, onePerHost bool) *Kite {
	hostname, _ := os.Hostname()
	serviceUniqueName := "kite-" + name + "-" + strconv.Itoa(os.Getpid()) + "|" + strings.Replace(hostname, ".", "_", -1)
	if onePerHost {
		serviceUniqueName = "kite-" + name + "|" + strings.Replace(hostname, ".", "_", -1)
	}

	return &Kite{
		Name:              name,
		Handlers:          make(map[string]Handler),
		ServiceUniqueName: serviceUniqueName,
	}
}

func (k *Kite) Handle(method string, concurrent bool, callback func(args *dnode.Partial, channel *Channel) (interface{}, error)) {
	k.Handlers[method] = Handler{concurrent, callback}
}

func (k *Kite) Run() {
	consumeConn := amqputil.CreateConnection("kite-" + k.Name)
	defer consumeConn.Close()

	publishConn := amqputil.CreateConnection("kite-" + k.Name)
	defer publishConn.Close()

	publishChannel := amqputil.CreateChannel(publishConn)
	defer publishChannel.Close()

	consumeChannel := amqputil.CreateChannel(consumeConn)

	amqputil.JoinPresenceExchange(consumeChannel, "services-presence", "kite", "kite-"+k.Name, k.ServiceUniqueName, k.LoadBalancer != nil)

	stream := amqputil.DeclareBindConsumeQueue(consumeChannel, "fanout", k.ServiceUniqueName, "", true)
	go k.startRouting(stream, publishChannel)

	// listen to an external control channel
	controlChannel := amqputil.CreateChannel(consumeConn)
	defer controlChannel.Close()

	controlStream := amqputil.DeclareBindConsumeQueue(controlChannel, "fanout", "control", "", true)
	controlRouting(controlStream) // blocking
}

func controlRouting(stream <-chan amqp.Delivery) {
	for msg := range stream {
		switch msg.RoutingKey {
		// those are temporary here
		// and should not be here
		case "control.suspendVM":
			var control Control
			err := json.Unmarshal(msg.Body, &control)
			if err != nil || control.HostnameAlias == "" {
				log.Err("Invalid control message.", string(msg.Body))
				continue
			}

			v, err := modelhelper.GetVM(control.HostnameAlias)
			if err != nil {
				log.Err("vm not found '%s'", control.HostnameAlias)
				continue
			}

			vm := virt.VM(v)
			if err := vm.Stop(); err != nil {
				log.Err("could not stop vm '%s'", control.HostnameAlias)
				continue
			}
		}
	}
}

func (k *Kite) startRouting(stream <-chan amqp.Delivery, publishChannel *amqp.Channel) {
	changeClientsGauge := lifecycle.CreateClientsGauge()
	log.RunGaugesLoop()

	timeoutChannel := make(chan string)

	routeMap := make(map[string](chan<- []byte))
	defer func() {
		for _, route := range routeMap {
			close(route)
		}
	}()

	for {
		select {
		case message, ok := <-stream:
			if !ok {
				return
			}

			switch message.RoutingKey {
			case "auth.join":
				log.Debug("auth.join", message)

				var channel Channel
				err := json.Unmarshal(message.Body, &channel)
				if err != nil || channel.Username == "" || channel.RoutingKey == "" {
					log.Err("Invalid auth.join message.", message.Body)
					continue
				}

				if _, found := routeMap[channel.RoutingKey]; found {
					// log.Warn("Duplicate auth.join for same routing key.")
					continue
				}
				route := make(chan []byte, 1024)
				routeMap[channel.RoutingKey] = route

				go func() {
					defer log.RecoverAndLog()
					defer channel.Close()

					changeClientsGauge(1)
					log.Debug("Client connected: " + channel.Username)
					defer func() {
						changeClientsGauge(-1)
						log.Debug("Client disconnected: " + channel.Username)
					}()

					d := dnode.New()
					defer d.Close()
					d.OnRootMethod = func(method string, args *dnode.Partial) {
						defer log.RecoverAndLog()

						if method == "ping" {
							d.Send("pong")
							return
						}
						if method == "pong" {
							return
						}

						var partials []*dnode.Partial
						err := args.Unmarshal(&partials)
						if err != nil {
							panic(err)
						}

						var options struct {
							WithArgs *dnode.Partial
						}
						err = partials[0].Unmarshal(&options)
						if err != nil {
							panic(err)
						}
						var resultCallback dnode.Callback
						err = partials[1].Unmarshal(&resultCallback)
						if err != nil {
							panic(err)
						}

						handler, found := k.Handlers[method]
						if !found {
							resultCallback(CreateErrorObject(&UnknownMethodError{Method: method}), nil)
							return
						}

						execHandler := func() {
							defer func() {
								if err := recover(); err != nil {
									log.LogError(err, 1, channel.Username, channel.CorrelationName)
									time.Sleep(time.Second) // penalty for avoiding that the client rapidly sends the request again on error
									resultCallback(CreateErrorObject(&InternalKiteError{}), nil)
								}
							}()

							result, err := handler.Callback(options.WithArgs, &channel)
							if b, ok := result.([]byte); ok {
								result = string(b)
							}

							if err != nil {
								if _, ok := err.(*WrongChannelError); ok {
									if err := publishChannel.Publish("broker", channel.RoutingKey+".cycleChannel", false, false, amqp.Publishing{Body: []byte("null")}); err != nil {
										log.LogError(err, 0)
									}
									return
								}

								resultCallback(CreateErrorObject(err), result)
								return
							}

							resultCallback(nil, result)
						}

						if handler.Concurrent {
							go execHandler()
							return
						}

						execHandler()
					}

					go func() {
						defer log.RecoverAndLog()
						for data := range d.SendChan {
							log.Debug("Write", channel.RoutingKey, data)
							if err := publishChannel.Publish("broker", channel.RoutingKey, false, false, amqp.Publishing{Body: data}); err != nil {
								log.LogError(err, 0)
							}
						}
					}()

					d.Send("ready", k.ServiceUniqueName)

					pingAlreadySent := false
					for {
						select {
						case message, ok := <-route:
							if !ok {
								return
							}
							func() {
								defer func() {
									if err := recover(); err != nil {
										log.LogError(err, 1, channel.Username, channel.CorrelationName, message)
									}
								}()
								log.Debug("Read", channel.RoutingKey, message)
								d.ProcessMessage(message)
							}()
							pingAlreadySent = false
						case <-time.After(5 * time.Minute):
							if pingAlreadySent {
								timeoutChannel <- channel.RoutingKey
								break
							}
							d.Send("ping")
							pingAlreadySent = true
						}
					}
				}()

			case "auth.leave":
				// ignored, session end is handled by ping/pong timeout

			case "auth.who":
				var client struct {
					Username           string `json:"username"`
					RoutingKey         string `json:"routingKey"`
					CorrelationName    string `json:"correlationName"`
					DeadService        string `json:"deadService"`
					ReplyExchange      string `json:"replyExchange"`
					ServiceGenericName string `json:"serviceGenericName"`
					ServiceUniqueName  string `json:"serviceUniqueName"` // used only for response
				}
				err := json.Unmarshal(message.Body, &client)
				if err != nil || client.Username == "" || client.RoutingKey == "" || client.CorrelationName == "" {
					log.Err("Invalid auth.who message.", message.Body)
					continue
				}
				if k.LoadBalancer == nil {
					log.Err("Got auth.who without having a load balancer.", message.Body)
					continue
				}

				client.ServiceUniqueName = k.LoadBalancer(client.CorrelationName, client.Username, client.DeadService)
				response, err := json.Marshal(client)
				if err != nil {
					log.LogError(err, 0)
					continue
				}
				if client.ReplyExchange == "" { // backwards-compatibility
					client.ReplyExchange = "auth"
				}
				if err := publishChannel.Publish(client.ReplyExchange, "kite.who", false, false, amqp.Publishing{Body: response}); err != nil {
					log.LogError(err, 0)
				}

			default:
				route, found := routeMap[message.RoutingKey]
				if found {
					select {
					case route <- message.Body:
						// successful
					default:
						close(route)
						delete(routeMap, message.RoutingKey)
						log.Warn("Dropped client because of message buffer overflow.")
					}
				}
			}

		case routingKey := <-timeoutChannel:
			route, found := routeMap[routingKey]
			if found {
				close(route)
				delete(routeMap, routingKey)
			}
		}
	}

}

func (channel *Channel) OnDisconnect(f func()) {
	channel.onDisconnect = append(channel.onDisconnect, f)
}

func (channel *Channel) Close() {
	channel.Alive = false
	for _, f := range channel.onDisconnect {
		f()
	}
	channel.onDisconnect = nil
}
