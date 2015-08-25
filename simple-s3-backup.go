package main

import (
	"bufio"
	"errors"
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
	fileList []string
	upload   chan string
	done     chan string
	bucket   *s3.Bucket
	wg       sync.WaitGroup
	bar      *pb.ProgressBar
}

func main() {

	// declare flags
	regionPtr := flag.String("region", "us-east-2", "AWS Region(default: us-east-2)")
	bucketPtr := flag.String("bucket", "backup", "Bucket path you wish to upload")
	srcPtr := flag.String("src", os.Getenv("CWD"), "src folder of files (default this folder)")
	keyPtr := flag.String("key ", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS_ACCESS_KEY_ID")
	secretPtr := flag.String("secret", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS_SECRET_ACCESS_KEY")

	os.Setenv("AWS_ACCESS_KEY_ID", *keyPtr)
	os.Setenv("AWS_SECRET_ACCESS_KEY_ID", *secretPtr)

	// Authenticate
	// AWSAuth, err := aws.EnvAuth()
	// region := aws.USWest2
	svc := s3.New(&aws.Config{Region: aws.String(*regionPtr)})

	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) == 1 {
		log.Fatal(errors.New(fmt.Sprintf("Did not receive src or dest path")))
	}
	if len(os.Args) == 2 {
		log.Fatal(errors.New(fmt.Sprintf("Did not receive a dest path")))
	}

	// @1 path
	// @2 bucket name
	// @3 zone
	// @4 key
	// @5 secret
	//
	//

	connection := s3.New(AWSAuth, region)

	s4 := StupidSimpleS3Backup{
		fileList: make([]string, 0),
		upload:   make(chan string),
		done:     make(chan string),
		bucket:   connection.Bucket(*bucketPtr),
	}

	s4.fileList = GenFileList(os.Args[1])
	count := len(s4.fileList)
	s4.bar = pb.StartNew(count - 1)
	s4.bar.ShowTimeLeft = false
	s4.bar.Format("[⚡- ]")

	s4.wg.Add(len(s4.fileList) - 1)

	// Start 2 go routines  that spin off channels for handling parellel file uploads
	// and completions

	go s4.pushFiles(os.Args[1])
	go s4.fileManager()

	s4.wg.Wait()

	close(s4.done)
	close(s4.upload)
	s4.bar.FinishPrint("The End!")
}

func (s4 *StupidSimpleS3Backup) pushFiles(prefix string) error {
	for {
		select {
		case path := <-s4.upload:
			go func(p string) {
				// log.Println("Received file", p)
				defer s4.wg.Done()

				file, err := os.Open(p)
				if err != nil {
					log.Panic(err)
				}
				defer file.Close()

				fileInfo, _ := file.Stat()
				var size int64 = fileInfo.Size()
				bytes := make([]byte, size)

				// read into buffer
				buffer := bufio.NewReader(file)
				_, err = buffer.Read(bytes)

				filetype := http.DetectContentType(bytes)
				fileName := strings.TrimPrefix(file.Name(), prefix+"/")
				err = s4.bucket.Put("n3w/"+fileName, bytes, filetype, s3.ACL("public-read"), s3.Options{})
				if err != nil {
					log.Panic(err)
				}

				s4.done <- fileName
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

func (s4 *StupidSimpleS3Backup) fileManager() {
	index := 0
	numFiles := len(s4.fileList)

	if numFiles < limit {
		limit = numFiles
	}
	// start go func to
	go func() {
		for {
			select {
			case name := <-s4.done:

				if DEBUG {
					log.Println("Received done for", name)
					log.Printf("%+v", s4.wg)
				}

				// Previous file completed uploading
				index++
				s4.bar.Increment()
				if index < numFiles {
					s4.upload <- s4.fileList[index]
				}
			}
		}
	}()

	for ; index < limit; index++ {
		s4.upload <- s4.fileList[index]
	}

}
