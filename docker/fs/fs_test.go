package fs_test

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var dataDirPath string
var nOps int
var nDummyFiles int

func init() {
	flag.StringVar(&dataDirPath, "data_dir_path", "/data", "path to data dir")
	flag.IntVar(&nOps, "n_ops", 1000, "Number of ops")
	flag.IntVar(&nDummyFiles, "n_dummy_files", 0, "Number of dummy files")
}

func TestCompile(t *testing.T) {
}

func writeAtomic(path string) error {
	const (
		dirPerms  = 0700
		filePerms = 0600
	)
	tempPath := path + ".tmp"

	if err := os.MkdirAll(filepath.Dir(path), dirPerms); err != nil {
		return err
	}
	fh, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerms)
	if err != nil {
		return err
	}
	contents := make([]byte, 400)
	if _, err := fh.Write(contents); err != nil {
		fh.Close()
		os.Remove(tempPath)
		return err
	}
	if err := fh.Sync(); err != nil {
		fh.Close()
		os.Remove(tempPath)
		return err
	}
	if err := fh.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}

func TestWriteAtomic(t *testing.T) {
	for i := 0; i < nDummyFiles; i++ {
		err := writeAtomic(filepath.Join(dataDirPath, strconv.Itoa(i+nOps)))
		if !assert.Nil(t, err, "Err writeAtomic: %v", err) {
			return
		}
	}
	start := time.Now()
	defer func(start time.Time) {
		log.Printf("elapsed:%v ops:%v time/op:%0.4fs", time.Since(start), nOps, time.Since(start).Seconds()/float64(nOps))
	}(start)
	for i := 0; i < nOps; i++ {
		err := writeAtomic(filepath.Join(dataDirPath, strconv.Itoa(i)))
		if !assert.Nil(t, err, "Err writeAtomic: %v", err) {
			return
		}
	}
}
