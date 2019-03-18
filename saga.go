package xaga

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
)

const (
	//StateBegin state begin saga operation
	StateBegin = "SAGA_BEGIN"
	//StateEnd state end saga operation
	StateEnd = "SAGA_END"
	//StateFailed state failed for saga operation
	StateFailed = "SAGA_FAILED"
	//StateSuccess state success for sage
	StateSuccess = "SAGA_SUCCESS"
)

const (
	//FmtStateBegin format for state begin
	FmtStateBegin = "STATE_%s_BEGIN"
	//FmtStateSuccess format for state success
	FmtStateSuccess = "STATE_%s_SUCCESS"
	//FmtStateFailed format for state failed
	FmtStateFailed = "STATE_%s_FAILED"
	//FmtStateCompensate format for state compensation
	FmtStateCompensate = "STATE_%s_COMPENSATE"
)

//SagaFunc anonymous function for saga handler
type SagaFunc func(ctx context.Context) error

//TransactFunc anonymous function for saga transaction handler
type TransactFunc func(ctx context.Context) error

//Log as saga log structure
type Log struct {
	State          string   `json:"state"`
	Origin         string   `json:"origin"`
	Message        *Message `json:"message,omitempty"`
	IsCompensation bool     `json:"is_compensation,omitempty"`
}

//Message as saga log's message structure
type Message struct {
	SagaID string      `json:"saga_id,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type Transact struct {
	State string
	Data  interface{} `json:"data"`
}

type PayLoad struct {
	SagaID string
	Logs   []*Log
}

type Saga interface {
	PayLoad() *PayLoad
	Do(ctx context.Context, sf SagaFunc) error
	Transact(ctx context.Context, state string, input interface{}, cf TransactFunc) error
}

type saga struct {
	id        string
	topic     string
	abort     bool
	logs      []*Log
	transacts []*Transact
	redis     *Redis
}

func (s *saga) Do(ctx context.Context, sf SagaFunc) (err error) {
	s.push(&Log{
		State: StateBegin,
	})
	defer func() {
		if err != nil {
			s.push(&Log{
				State: StateFailed,
				Message: &Message{
					Error: err.Error(),
				},
			})
			s.abort = true
		} else {
			s.push(&Log{
				State: StateSuccess,
			})
		}
		s.push(&Log{
			State: StateEnd,
		})
		go s.publish()
	}()
	err = sf(ctx)
	return
}

func (s *saga) Transact(ctx context.Context, state string, input interface{}, tf TransactFunc) (err error) {
	if s.abort {
		return
	}
	s.push(&Log{
		State: fmt.Sprintf(FmtStateBegin, state),
	})
	s.addTx(&Transact{
		Data:  input,
		State: state,
	})
	defer func() {
		if err != nil {
			s.push(&Log{
				State: fmt.Sprintf(FmtStateFailed, state),
				Message: &Message{
					Error: err.Error(),
				},
			})
			s.compensateLogs()
		} else {
			s.push(&Log{
				State: fmt.Sprintf(FmtStateSuccess, state),
			})
		}
	}()
	err = tf(ctx)
	return
}

func (s *saga) PayLoad() *PayLoad {
	return &PayLoad{
		SagaID: s.id,
		Logs:   s.logs,
	}
}

func (s *saga) push(log *Log) {
	s.logs = append(s.logs, log)
}

func (s *saga) addTx(tx *Transact) {
	s.transacts = append(s.transacts, tx)
}

func (s *saga) compensateLogs() {
	for i := len(s.transacts) - 1; i >= 0; i-- {
		tx := s.transacts[i]
		s.push(&Log{
			Origin: tx.State,
			State:  fmt.Sprintf(FmtStateCompensate, tx.State),
			Message: &Message{
				Data: tx.Data,
			},
			IsCompensation: true,
		})
	}
}

func (s *saga) publish() {
	bytes, _ := json.Marshal(s.PayLoad())
	conn := s.redis.Pool.Get()
	ret, err := conn.Do("PUBLISH", s.topic, string(bytes))
	defer conn.Close()
	log.Println("ret=", ret, ",err=", err)
}
