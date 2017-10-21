package peekaboos3

import (
	"log"
	"os"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"fmt"
)

func getSession() (*session.Session, error) {
    return session.NewSession(&aws.Config{
        Region: aws.String("ap-southeast-1"),
        Credentials: credentials.NewSharedCredentials("", "peekaboo"),
    })
}

func DownloadFile(bucket string, key string, path string) error {
	sess, err := getSession()
	if err != nil {
		log.Println("could not get session, stopping download")
		return err
	}
    downloader := s3manager.NewDownloader(sess)
	f, err := os.Create(path)
	if err != nil {
		log.Println("could not create file, stopping download")
		return err
	}
    _ , err = downloader.Download(f, &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        log.Println(err)
	}
	return err
}

func CopyFile(bucket string, key string, newKey string) error {

	sess, err := getSession()
	if err != nil {
		return err
	}
	svc := s3.New(sess)

	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(fmt.Sprintf("/%s/%s", bucket, key)),
		Key:        aws.String(newKey),
	}
	
	_, err = svc.CopyObject(copyInput)
	if err != nil {
		return err
	}
	return nil
}

func DeleteFile(bucket string, key string) error {

	err := CopyFile(bucket, key, fmt.Sprintf("deleted/%s", key))
	if err != nil {
		return err
	}
	sess, err := getSession()
	if err != nil {
		return err
	}
	svc := s3.New(sess)
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	
	_, err = svc.DeleteObject(deleteInput)
	if err != nil {
		return err
	}
	return nil
}

func UploadFile(bucket string, key string, path string) error {
	sess, err := getSession()
	if err != nil {
		log.Println("could not get session, stopping upload")
		return err
	}
	uploader := s3manager.NewUploader(sess)
	f, err := os.Open(path)
	if err != nil {
		log.Println("could not open file, stopping upload")
		return err
	}
    _ , err = uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body: f,
    })
    if err != nil {
        log.Println(err)
	}
	return err
}