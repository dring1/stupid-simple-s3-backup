package s5

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	limit = 25
	DEBUG = false
)

type StupidSimpleS3Backup struct {
	fileList  []string
	FileCount int
	upload    chan string
	done      chan string
	Bucket    string
	Dest      string
	wg        sync.WaitGroup
	svc       *s3.S3
	src       string
}

func New(src string, dest string, bucket string, key string, secret string, region string, debug bool) *StupidSimpleS3Backup {
	DEBUG = debug
	creds := credentials.NewStaticCredentials(key, secret, "")
	_, err := creds.Get()

	if err != nil {
		log.Println(err)
		log.Println("Authentication failed.")
		os.Exit(1)
	}

	fileList := GenFileList(src)

	return &StupidSimpleS3Backup{
		fileList:  fileList,
		FileCount: len(fileList),
		upload:    make(chan string),
		done:      make(chan string),
		Bucket:    bucket,
		Dest:      dest,
		src:       src,
		svc: s3.New(&aws.Config{Region: aws.String(region),
			Credentials:      creds,
			S3ForcePathStyle: aws.Bool(true)}),
	}

}

func (s5 *StupidSimpleS3Backup) Run(cb func()) {

	s5.wg.Add(s5.FileCount)
	go s5.FileDispatcher(cb)
	go s5.FileManager()

	s5.wg.Wait()

	close(s5.done)
	close(s5.upload)
}

func (s5 *StupidSimpleS3Backup) FileDispatcher(cb func()) error {
	for {
		select {
		case fpath := <-s5.upload:
			go s5.FileUpload(fpath, cb)
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

func (s5 *StupidSimpleS3Backup) FileManager() {
	index := limit - 1

	if s5.FileCount < limit {
		limit = s5.FileCount
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

				index++
				// if the index is less than the file count
				// files still remain in the pool
				if index < s5.FileCount {
					s5.upload <- s5.fileList[index]
				}
			}
		}
	}()

	// kick off the
	for i := 0; i < limit; i++ {
		s5.upload <- s5.fileList[i]
	}

}

func (s5 *StupidSimpleS3Backup) FileUpload(fp string, cb func()) {
	defer s5.wg.Done()
	defer cb()

	file, err := os.Open(fp)
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
	fileName := strings.TrimPrefix(file.Name(), s5.src+"/")

	if DEBUG {
		log.Println("Evaluating file:", s5.Dest+"/"+fileName)
	}

	params := &s3.PutObjectInput{
		Bucket:        aws.String(s5.Bucket),                // Required
		Key:           aws.String(s5.Dest + "/" + fileName), // Required
		Body:          bytes.NewReader(payload),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(filetype),
	}
	resp, err := s5.svc.PutObject(params)

	if err != nil {
		log.Println(err)
		return
	}
	if err != nil {
		log.Panic(err)
	}
	if DEBUG {
		log.Printf("response: %+v\n", resp)
	}
	s5.done <- fileName
}
