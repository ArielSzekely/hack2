package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mattetti/filebuffer"
)

const (
	DUR   time.Duration = 25 * time.Second
	MB                  = 1 << 20
	BUFSZ               = 6 * MB
)

type Event struct {
	ObjPath string `json:"obj_path"`
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

func s3perf(s3cli *s3.S3, key string) *Result {
	bucket := "9ps3"
	params := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	result, err := s3cli.CreateMultipartUpload(params)
	if err != nil {
		return NewResult("", fmt.Errorf("Error CreateMultipartUpload: %v", err))
	}
	uploadID := *result.UploadId

	b := make([]byte, BUFSZ)
	for i := range b {
		b[i] = 'g'
	}
	nbyte := 0
	start := time.Now()
	completedParts := []*s3.CompletedPart{}
	for partNumber := int64(1); time.Since(start) < DUR; partNumber++ {
		buf := filebuffer.New(b)
		buf.Seek(0, io.SeekStart)
		uploadParams := &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(uploadID),
			Body:       buf,
			PartNumber: aws.Int64(partNumber),
		}
		result, err := s3cli.UploadPart(uploadParams)
		if err != nil {
			return NewResult("", fmt.Errorf("Error UploadPart: %v", err))
		}
		completedParts = append(completedParts, &s3.CompletedPart{
			ETag:       result.ETag,
			PartNumber: aws.Int64(partNumber),
		})
		nbyte += len(b)
	}

	completeParams := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	_, err = s3cli.CompleteMultipartUpload(completeParams)
	if err != nil {
		return NewResult("", fmt.Errorf("Error CompleteMultipartUpload: %v", err))
	}
	mb := float64(nbyte) / float64(MB)
	tpt := mb / time.Since(start).Seconds()
	return NewResult(fmt.Sprintf("%.2fMB/s", tpt), nil)
}

func HandleRequest(ctx context.Context, event *Event) (*string, error) {
	log.Printf("Handle request: %s", event.ObjPath)
	defer log.Printf("Handle request done: %s", event.ObjPath)
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	s3cli := s3.New(sess)
	res := s3perf(s3cli, event.ObjPath)
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
		HandleRequest(context.TODO(), &Event{
			ObjPath: "s3perf-test-obj",
		})
	}
}
