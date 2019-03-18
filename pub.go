package xaga

func NewProducer(topic string, redis *Redis) Producer {
	return &producer{
		topic: topic,
		redis: redis,
	}
}

type Producer interface {
	New(sagaID string) Saga
}

type producer struct {
	topic string
	redis *Redis
}

//New creates saga instance
//args:
//	sagaID saga identification
//returns:
//	Saga instance
func (p *producer) New(sagaID string) Saga {
	return &saga{
		id:        sagaID,
		logs:      make([]*Log, 0),
		transacts: make([]*Transact, 0),
		redis:     p.redis,
		topic:     p.topic,
	}
}
