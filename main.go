package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/dring1/simple-s3-upload/stupid-simple-s3-backup"
)

// This commit value will be assigned by the ld flag at compile time
var (
	timeStamp bool
	debug     bool
)

func main() {
	// declare flags
	regionPtr := flag.String("region", "us-east-2", "AWS Region(default: us-east-2)")
	bucketPtr := flag.String("bucket", "backup", "Bucket path you wish to upload")
	srcPtr := flag.String("src", os.Getenv("PWD"), "src folder of files (default this folder)")
	keyPtr := flag.String("key", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS_ACCESS_KEY_ID")
	secretPtr := flag.String("secret", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS_SECRET_ACCESS_KEY")
	destPtr := flag.String("dest", "new", "Destination path of root directory where you wish the contents to go in the bucket, \nif your connection continues to timeout try lowering it")
	limitPtr := flag.Int("limit", "4", "Number of concurrent uploads")

	flag.BoolVar(&timeStamp, "timestamp", false, "append commitstamp to the destination the commit")
	flag.BoolVar(&debug, "debug", false, "enable debug mode")

	flag.Parse()

	p := []string{
		*destPtr,
	}

	if timeStamp {
		p = append(p, strconv.FormatInt(time.Now().Unix(), 10))
	}

	dest := strings.Join(p, ".")
	fmt.Println(dest)
	simpleS3 := s5.New(*srcPtr, dest, *bucketPtr, *keyPtr, *secretPtr, *regionPtr, *limitPtr, debug)
	bar := pb.StartNew(simpleS3.FileCount)
	bar.ShowTimeLeft = false
	bar.Format("[⚡- ]")

	simpleS3.Run(func() {
		bar.Increment()
	})
	bar.FinishPrint(fmt.Sprintf("Completed Uploading to %s/%s!", simpleS3.Bucket, simpleS3.Dest))
}
