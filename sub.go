package xaga

import (
	"encoding/json"

	"github.com/gomodule/redigo/redis"
)

func NewConsumer(topic string, r *Redis) Consumer {
	return &consumer{
		r:     r,
		topic: topic,
		echan: make(chan error, 1),
		schan: make(chan bool, 1),
		funcs: make(map[string]CompensationFunc, 0),
	}
}

type CompensationFunc func(message *Message) error

type Consumer interface {
	RegisterCompensation(state string, cf CompensationFunc)
	Run()
	Stop()
	Err() <-chan error
}

type consumer struct {
	r     *Redis
	topic string
	funcs map[string]CompensationFunc
	echan chan error
	schan chan bool
}

func (c *consumer) RegisterCompensation(state string, cf CompensationFunc) {
	if _, ok := c.funcs[state]; !ok {
		c.funcs[state] = cf
	}
}

func (c *consumer) Run() {
	rc := c.r.Pool.Get()
	psc := redis.PubSubConn{
		Conn: rc,
	}
	if err := psc.PSubscribe(c.topic); err != nil {
		c.echan <- err
		return
	}
	for {
		select {
		case <-c.schan:
			rc.Close()
			psc.Close()
			return
		default:
			switch m := psc.Receive().(type) {
			case redis.Message:
				go c.process(m.Data)
			}
		}
	}
}

func (c *consumer) Err() <-chan error {
	return c.echan
}

func (c *consumer) Stop() {
	c.schan <- true
}

func (c *consumer) process(bytes []byte) {
	var p PayLoad
	json.Unmarshal(bytes, &p)
	for _, l := range p.Logs {
		if l.IsCompensation {
			if f, ok := c.funcs[l.Origin]; ok {
				l.Message.SagaID = p.SagaID
				f(l.Message)
			}
		}
	}
}
