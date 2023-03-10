package EventBus2

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
)

const (
	// PublishService - Client service method
	PublishService = "ClientService.PushEvent"
)

// ClientArg - object containing event for client to publish locally
type ClientArg struct {
	Args  []interface{}
	Topic string
}

// Client - object capable of subscribing to a remote event bus
type Client struct {
	eventBus Bus
	address  string
	path     string
	service  *ClientService
}

// NewClient - create a client object with the address and server path
func NewClient(address, path string, eventBus Bus) *Client {
	client := new(Client)
	client.eventBus = eventBus
	client.address = address
	client.path = path
	client.service = &ClientService{client, &sync.WaitGroup{}, false}
	return client
}

// Helper function - Trim string prior to the colon, if found
func trimStringBeforeColon(s string) string {
	if idx := strings.Index(s, ":"); idx != -1 {
		return s[idx:]
	}
	return s
}

// EventBus - returns the underlying event bus
func (client *Client) EventBus() Bus {
	return client.eventBus
}

func (client *Client) doSubscribe(topic string, fn interface{}, serverAddr, serverPath string, subscribeType SubscribeType) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Server not found -", r)
		}
	}()

	rpcClient, err := rpc.DialHTTPPath("tcp", serverAddr, serverPath)
	defer func(rpcClient *rpc.Client) {
		err := rpcClient.Close()
		if err != nil {
			err = fmt.Errorf("dial close failed: %v", err)
		}
	}(rpcClient)
	if err != nil {
		err = fmt.Errorf("dialing: %v", err)
	}
	args := &SubscribeArg{client.address, client.path, PublishService, subscribeType, topic}
	reply := new(bool)
	err = rpcClient.Call(RegisterService, args, reply)
	if err != nil {
		err = fmt.Errorf("register error: %v", err)
	}
	if *reply {
		err := client.eventBus.Subscribe(topic, fn)
		if err != nil {
			return
		}
	}
}

// Subscribe subscribes to a topic in a remote event bus
func (client *Client) Subscribe(topic string, fn interface{}, serverAddr, serverPath string) {
	client.doSubscribe(topic, fn, serverAddr, serverPath, Subscribe)
}

// SubscribeOnce subscribes once to a topic in a remote event bus
func (client *Client) SubscribeOnce(topic string, fn interface{}, serverAddr, serverPath string) {
	client.doSubscribe(topic, fn, serverAddr, serverPath, SubscribeOnce)
}

// Start - starts the client service to listen to remote events
func (client *Client) Start() error {
	var err error
	service := client.service
	if !service.started {
		server := rpc.NewServer()
		err := server.Register(service)
		if err != nil {
			return err
		}
		server.HandleHTTP(client.path, "/debug"+client.path)
		l, err := net.Listen("tcp", trimStringBeforeColon(client.address))
		if err == nil {
			service.wg.Add(1)
			service.started = true
			go func() {
				err := http.Serve(l, nil)
				if err != nil {
					return
				}
			}()
		}
	} else {
		err = errors.New("client service already started")
	}
	return err
}

// Stop - signal for the service to stop serving
func (client *Client) Stop() {
	service := client.service
	if service.started {
		service.wg.Done()
		service.started = false
	}
}

// ClientService - service object listening to events published in a remote event bus
type ClientService struct {
	client  *Client
	wg      *sync.WaitGroup
	started bool
}

// PushEvent - exported service to listening to remote events
func (service *ClientService) PushEvent(arg *ClientArg, reply *bool) error {
	service.client.eventBus.Publish(arg.Topic, arg.Args...)
	*reply = true
	return nil
}
