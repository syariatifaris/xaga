package main

import (
	"context"
	"errors"
	"log"
	"time"

	"encoding/json"
	"github.com/syariatifaris/xaga"
)

func main() {
	topic := "saga_test"
	//create redis
	r, err := xaga.NewRedis("localhost:6379")
	if err != nil {
		log.Fatalln("unable to initiate redis", err)
	}
	//create producer
	p := xaga.NewProducer(topic, r)
	//do sample saga transaction
	payload, err := do(p)
	if err != nil {
		log.Println(err)
	}
	time.Sleep(time.Second / 2)
	bytes, err := json.Marshal(payload)
	log.Println(string(bytes))
}

func do(p xaga.Producer) (*xaga.PayLoad, error) {
	saga := p.New("123456")
	err := saga.Do(context.Background(), func(ctx context.Context) error {
		//first transaction
		err := saga.Transact(ctx, "state_transaction_1", "message", func(ctx context.Context) error {
			log.Println("do something in transaction 1")
			return nil
		})
		if err != nil {
			return err
		}
		//second transaction
		err = saga.Transact(ctx, "state_transaction_2", "message 2", func(ctx context.Context) error {
			log.Println("do something in transaction 2")
			return errors.New("error in tx 2")
		})
		return err
	})
	return saga.PayLoad(), err
}
