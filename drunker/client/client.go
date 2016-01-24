package client

import (
	"encoding/json"
	"log"
	"strconv"
	"time"
	"go-concurrency/messages"
	"go-concurrency/database"
)

// this client is an aggregation of one DBClient (Redis for the moment)
// and one broker client (Nsq here)
// will receive order created from some producer, register it on the Db
// and send it to the waiters through the broker
type Client struct {
	redisCl        database.DbClient
	brokerProducer message.BrokerProducer
	topic          string
	stopChan       chan bool
	frequency      int
	ttl 		   int
}

// create and start a new client with one DataBase client, one broker client
// the topic to use for the broker and the number of order producer to launch
func StartClient(dbClient database.DbClient, brokerProducer message.BrokerProducer, topic string, f int, ttl int) (c *Client, err error) {
	c = new(Client)
	c.redisCl = dbClient
	c.brokerProducer = brokerProducer
	c.topic = topic
	c.stopChan = make(chan bool, 1)
	c.frequency = f
	c.ttl = ttl
	go c.listen()
	return
}

func (c *Client) listen() {
	for {
		select {
		case <-c.stopChan:
			log.Println("The client is stopping")
			return
		default:
			o := message.NewOrder(message.NextBeverageType())
			json, _ := json.Marshal(o)
			log.Printf("json:\n%s",json)
			errR := c.redisCl.Set(strconv.Itoa(int(o.Id)), json, time.Duration(c.ttl))
			if errR != nil {
				log.Printf("error during redis registration: %v", errR)
			} else {
				errB := c.brokerProducer.Publish(c.topic, json)
				if errB != nil {
					log.Printf("error during broker registration: %v", errB)
				}
			}
		time.Sleep(time.Duration(time.Millisecond * time.Duration(c.frequency)))
		}
	}
}

func (c *Client) StopClient() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovery on some error while trying to close channels : %f", r)
		}
	}()
	if c.stopChan != nil {
		c.stopChan <- true
	}
	return
}
