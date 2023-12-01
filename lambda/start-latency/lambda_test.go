package lambda_test

import (
	"encoding/json"
	"flag"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"

	"lambda-start-latency/api"
)

var LOCAL bool

func init() {
	flag.BoolVar(&LOCAL, "local", false, "Run against local lambda container")
}

func TestLambdaLatency(t *testing.T) {
	log.Printf("Success!")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ev := &api.Event{
		CurTimeUsec: time.Now().UnixMicro(),
		SrvHTTP:     "x.x.x.x",
	}

	client := lambda.New(sess, &aws.Config{
		Region:   aws.String("us-east-1"),
		Endpoint: aws.String("127.0.0.1:9000"),
	})

	payload, err := json.Marshal(ev)
	if err != nil {
		log.Fatalf("Error marshalling lambda request: %v", err)
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String("lambda-start-latency"), Payload: payload})
	if err != nil {
		log.Fatalf("Error invoking lambda: %v", err)
	}
	assert.Equal(t, int(*result.StatusCode), 200, "Status code: %v", result.StatusCode)
	if *result.StatusCode != 200 {
		log.Fatalf("Bad return status %v, msg %v", result.StatusCode, result.Payload)
	}
	var res api.Result
	err = json.Unmarshal(result.Payload, &res)
	if err != nil {
		log.Fatalf("Error marshalling lambda request: %v", err)
	}
	log.Printf("E2e start latency: %vus", res.ElapsedUsec)
}
