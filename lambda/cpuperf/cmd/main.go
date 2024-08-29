package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	DUR    time.Duration = 25 * time.Second
	NUM_IT               = 1000000
)

type Event struct {
}

type Result struct {
	Throughput string `json:"throughput"`
	Err        string `json:"err"`
}

func NewResult(tpt string, err error) *Result {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return &Result{
		Throughput: tpt,
		Err:        errStr,
	}
}

func cpuperf() *Result {
	start := time.Now()
	i := uint64(0)
	for time.Since(start) < DUR {
		j := float64(1)
		for k := 0; k < NUM_IT; k++ {
			j = j*float64(k*k) + 1.0
		}
		i += NUM_IT
	}
	tpt := float64(i) / float64(1000000) / time.Since(start).Seconds()
	return NewResult(fmt.Sprintf("%.2fM iterations/s", tpt), nil)
}

func HandleRequest(ctx context.Context, event *Event) (*string, error) {
	log.Printf("Handle request: %s", event)
	defer log.Printf("Handle request done: %s", event)
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}
	res := cpuperf()
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("Error marshal json: %v", err)
	}
	message := string(b)
	log.Printf(message)
	return &message, nil
}

func main() {
	if os.Getenv("LOCAL_DEV") == "" {
		lambda.Start(HandleRequest)
	} else {
		res, err := HandleRequest(context.TODO(), &Event{})
		log.Printf("Write Res: %v\nErr:%v", *res, err)
	}
}
