package main

import (
	"context"
	"log"
	"os"
	"time"

	"encoding/json"
	"os/signal"
	"syscall"

	"github.com/syariatifaris/xaga"
)

func main() {
	topic := "saga_test"
	//create redis
	r, err := xaga.NewRedis("localhost:6379")
	if err != nil {
		log.Fatalln("unable to initiate redis", err)
	}
	//consumer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	//register compensation
	c := xaga.NewConsumer(topic, r)
	c.RegisterCompensation("state_transaction_1", compensateTx1)
	c.RegisterCompensation("state_transaction_2", compensateTx2)
	//listen for compensation event
	go func() {
		c.Run()
	}()
	select {
	case <-c.Err():
		log.Println("err", err)
	case <-ctx.Done():
		c.Stop()
		log.Println("timeout process")
	case <-term:
		c.Stop()
		log.Println("signal terminated detected")
	}
}

func compensateTx1(m *xaga.Message) error {
	bytes, _ := json.Marshal(m)
	log.Println("tx 1 message from saga log", string(bytes))
	return nil
}

func compensateTx2(m *xaga.Message) error {
	bytes, _ := json.Marshal(m)
	log.Println("tx 2 message from saga log", string(bytes))
	return nil
}
