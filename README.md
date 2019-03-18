# XAGA - Redis Powered Saga Helper

## How To

This library provide a helper to handle transaction in saga pattern, and handle its compensation transaction when error happen asynchronously as a rollback action.
It uses Redis Pub-Sub mechanism to publish the Saga Log to Redis. The consumer will listen to the event, and run the compensation for transaction error based on predefined stub.

## Redis Initialization

This library using `gomodule/redis` to create a redis connection.
```
rds, err := xaga.NewRedis("localhost:6379")
if err != nil {
    log.Fatalln("unable to initiate redis", err)
}
```

## Using Saga Transaction

This helper provide a wrapper for saga transaction.

### Create Saga Producer

The saga producer creates a saga instance as a set of transaction.
```
topic := "saga_trasaction"
p := xaga.NewProducer(topic, rds)
```

### Initiate a Transaction Set

For each transaction the client need to create a saga instance. Set of transaction should be wrapped inside the stub.

```
//each saga transaction need ID
saga := p.New("c3836eef-9885-4fc7-99b5-d5d10e4a9c42")
err := saga.Do(context.Background(), func(ctx context.Context) error {
    //first transaction
    err := saga.Transact(ctx, "STATE_1", "message", func(ctx context.Context) error {
        log.Println("Do something in STATE_1")
        return nil
    })
    if err != nil {
        return err
    }
    //second transaction
    err = saga.Transact(ctx, "STATE_2", "message 2", func(ctx context.Context) error {
        log.Println("Do something in STATE_2")
        return errors.New("error in tx 2")
    })
    return err
})
```

Each transaction (http call to API or database operation) which require a compensation transaction should be wrap inside `saga.Transact`. When error happens, it will register the event to redis pub-sub, and will be accepted on the consumer.

## Compensate Transaction

From above example it can be seen that 2nd transaction fails. Then, we expect to rollback 2nd transaction and 1st transaction. 

### Create Saga Consumer

The consumer will listen for the event on saga log.

```
topic := "saga_trasaction"
c := xaga.NewConsumer(topic, r)
```

### Register Compensation Transaction

```
c.RegisterCompensation("STATE_1", compensateTx1)
c.RegisterCompensation("STATE_2", compensateTx2)
```
Then we can create a callback to be registered

```
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
```

### Running Saga Consumer

Finally we can run Saga transaction like other handler


```
ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
defer cancel()
term := make(chan os.Signal, 1)
signal.Notify(term, os.Interrupt, syscall.SIGTERM)
//register compensation
c := xaga.NewConsumer(topic, r)
c.RegisterCompensation("STATE_1", compensateTx1)
c.RegisterCompensation("STATE_2", compensateTx2)
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
```
