package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cheggaaa/pb"
	"github.com/dring1/simple-s3-upload/stupid-simple-s3-backup"
)

func main() {

	// declare flags
	regionPtr := flag.String("region", "us-east-2", "AWS Region(default: us-east-2)")
	bucketPtr := flag.String("bucket", "backup", "Bucket path you wish to upload")
	srcPtr := flag.String("src", os.Getenv("PWD"), "src folder of files (default this folder)")
	keyPtr := flag.String("key", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS_ACCESS_KEY_ID")
	secretPtr := flag.String("secret", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS_SECRET_ACCESS_KEY")
	debugPtr := flag.Bool("debug", false, "enable debug mode")
	destPtr := flag.String("dest", "new", "Destination path of root directory where you wish the contents to go in the bucket")

	flag.Parse()

	simpleS3 := s5.New(*srcPtr, *destPtr, *bucketPtr, *keyPtr, *secretPtr, *regionPtr, *debugPtr)
	bar := pb.StartNew(simpleS3.FileCount)
	bar.ShowTimeLeft = false
	bar.Format("[⚡- ]")

	simpleS3.Run(func() {
		bar.Increment()
	})
	bar.FinishPrint(fmt.Sprintf("Completed Uploading to %s/%s!", simpleS3.Bucket, simpleS3.Dest))
}
