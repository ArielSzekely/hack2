package main

import (
	"bytes"
	"context"
	"flag"
	"image"
	"image/jpeg"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nfnt/resize"
)

var (
	N_ROUNDS    int
	IMG_DIM     int
	BUCKET_NAME string
	INPUT_NAME  string
	OUTPUT_NAME string
)

func init() {
	flag.IntVar(&N_ROUNDS, "n_rounds", 1, "Number of rounds of computation")
	flag.IntVar(&IMG_DIM, "img_dim", 160, "Output image dimention")
	flag.StringVar(&BUCKET_NAME, "bucket_name", "9ps3", "S3 bucket name")
	flag.StringVar(&INPUT_NAME, "input_name", "img-save/8.jpg", "Input file name")
	flag.StringVar(&OUTPUT_NAME, "output_name", "img-process-out", "Output file name")
}

//
// Crop picture <in> to <out>
//

func main() {
	log.Printf("imgresize: %v", os.Args)

	ip, err := NewImgProcess()
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	ip.Work()
	log.Printf("Time %v e2e resize: %v", os.Args, time.Since(start))
}

type ImgProcess struct {
	inputs []string
	clnt   *s3.Client
}

func NewImgProcess() (*ImgProcess, error) {
	ip := &ImgProcess{}
	log.Printf("Args {%v}", os.Args)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Err Load default AWS config: %v", err)
	}
	clnt := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	ip.clnt = clnt
	return ip, nil
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func (ip *ImgProcess) Work() {
	log.Printf("Resize %v/%v -> %v/%v dim %v", BUCKET_NAME, INPUT_NAME, BUCKET_NAME, OUTPUT_NAME, IMG_DIM)
	input := &s3.GetObjectInput{
		Bucket: &BUCKET_NAME,
		Key:    &INPUT_NAME,
	}
	start := time.Now()
	result, err := ip.clnt.GetObject(context.TODO(), input)
	if err != nil {
		log.Fatalf("Err GetObject: %v", err)
	}
	defer func() {
		if err := result.Body.Close(); err != nil {
			log.Fatalf("Err close input body: %v", err)
		}
	}()
	log.Printf("Time GetObject: %v", time.Since(start))
	img, err := jpeg.Decode(result.Body)
	if err != nil {
		log.Fatalf("Err decode err:%v", err)
	}
	var imgOut image.Image
	start = time.Now()
	for i := 0; i < N_ROUNDS; i++ {
		imgOut = resize.Resize(uint(IMG_DIM), uint(IMG_DIM), img, resize.Lanczos3)
	}
	log.Printf("Time resize: %v", time.Since(start))
	start = time.Now()
	outbuf := bytes.NewBuffer(nil)
	wrt := &nopWriteCloser{outbuf}
	jpeg.Encode(wrt, imgOut, nil)
	output := &s3.PutObjectInput{
		Bucket: &BUCKET_NAME,
		Key:    &OUTPUT_NAME,
		Body:   bytes.NewReader(outbuf.Bytes()),
	}
	if _, err := ip.clnt.PutObject(context.TODO(), output); err != nil {
		log.Fatalf("Err PutObject: %v", err)
	}
	log.Printf("Time Encode/Put: %v", time.Since(start))
}
