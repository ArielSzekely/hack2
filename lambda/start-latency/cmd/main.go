package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"lambda-start-latency/api"
)

func HandleRequest(ctx context.Context, event *api.Event) (*string, error) {
	invokeTime := time.UnixMicro(event.CurTimeUsec)
	elapsed := time.Since(invokeTime)
	res := &api.Result{
		ElapsedUsec: elapsed.Microseconds(),
	}
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
		HandleRequest(context.TODO(), &api.Event{
			CurTimeUsec: time.Now().UnixMicro(),
		})
	}
}
