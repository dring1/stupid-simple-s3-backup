package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cheggaaa/pb"
	// "github.com/goamz/goamz/aws"
	// "github.com/goamz/goamz/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	limit = 20
	DEBUG = false
)

type StupidSimpleS3Backup struct {
	fileList  []string
	fileCount int
	upload    chan string
	done      chan string
	bucket    string
	wg        sync.WaitGroup
	bar       *pb.ProgressBar
	svc       *s3.S3
}

func main() {

	// declare flags
	regionPtr := flag.String("region", "us-east-2", "AWS Region(default: us-east-2)")
	bucketPtr := flag.String("bucket", "backup", "Bucket path you wish to upload")
	srcPtr := flag.String("src", os.Getenv("PWD"), "src folder of files (default this folder)")
	keyPtr := flag.String("key ", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS_ACCESS_KEY_ID")
	secretPtr := flag.String("secret", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS_SECRET_ACCESS_KEY")
	debugPtr := flag.Bool("debug", false, "enable debug mode")

	os.Setenv("AWS_ACCESS_KEY_ID", *keyPtr)
	os.Setenv("AWS_SECRET_ACCESS_KEY_ID", *secretPtr)

	flag.Parse()

	DEBUG = *debugPtr
	// @1 path
	// @2 bucket name
	// @3 zone
	// @4 key
	// @5 secret

	fmt.Println(*bucketPtr, *srcPtr, *keyPtr, *secretPtr)
	s5 := StupidSimpleS3Backup{
		fileList: make([]string, 0),
		upload:   make(chan string),
		done:     make(chan string),
		bucket:   *bucketPtr,
		svc:      s3.New(&aws.Config{Region: aws.String(*regionPtr)}),
	}

	s5.fileList = GenFileList(*srcPtr)
	s5.fileCount = len(s5.fileList)
	s5.bar = pb.StartNew(s5.fileCount)
	s5.bar.ShowTimeLeft = false
	s5.bar.Format("[⚡- ]")

	s5.wg.Add(s5.fileCount)

	// Start 2 go routines  that spin off channels for handling parellel file uploads
	// and completions

	go s5.pushFiles(*srcPtr)
	go s5.fileManager()

	s5.wg.Wait()

	close(s5.done)
	close(s5.upload)
	s5.bar.FinishPrint("The End!")
}

func (s5 *StupidSimpleS3Backup) pushFiles(prefix string) error {
	for {
		select {
		case path := <-s5.upload:
			go func(p string) {
				// log.Println("Received file", p)
				defer s5.wg.Done()

				// OLD WAY
				file, err := os.Open(p)
				if err != nil {
					log.Panic(err)
				}
				defer file.Close()

				fileInfo, _ := file.Stat()
				var size int64 = fileInfo.Size()
				payload := make([]byte, size)

				// read into buffer
				buffer := bufio.NewReader(file)
				_, err = buffer.Read(payload)

				filetype := http.DetectContentType(payload)
				fileName := strings.TrimPrefix(file.Name(), prefix+"/")
				// err = s5.bucket.Put("n3w/"+fileName, bytes, filetype, s3.ACL("public-read"), s3.Options{})

				params := &s3.PutObjectInput{
					Bucket:        aws.String(s5.bucket), // Required
					Key:           aws.String(fileName),  // Required
					Body:          bytes.NewReader(payload),
					CacheControl:  aws.String("CacheControl"),
					ContentLength: aws.Int64(size),
					ContentType:   aws.String(filetype),
					Metadata: map[string]*string{
						"Key": aws.String("MetadataValue"), // Required
						// More values...
					},
				}
				resp, err := s5.svc.PutObject(params)

				if err != nil {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					// fmt.Println(err.Error())
					log.Panic(err)
					return
				}
				log.Println("response:", resp)
				if err != nil {
					log.Panic(err)
				}

				s5.done <- fileName
			}(path)

		}
	}
	return nil
}

func GenFileList(src string) []string {
	list := []string{}
	filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			list = append(list, path)
		}
		return nil
	})
	return list
}

func (s5 *StupidSimpleS3Backup) fileManager() {
	index := 0
	numFiles := len(s5.fileList)

	if numFiles < limit {
		limit = numFiles
	}
	// start go func to
	go func() {
		for {
			select {
			case name := <-s5.done:

				if DEBUG {
					log.Println("Received done for", name)
					log.Printf("%+v", s5.wg)
				}

				// Previous file completed uploading
				index++
				s5.bar.Increment()
				if index < numFiles {
					s5.upload <- s5.fileList[index]
				}
			}
		}
	}()

	for ; index < limit; index++ {
		s5.upload <- s5.fileList[index]
	}

}
