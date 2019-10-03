package storage

import (
	"fmt"
	"mime/multipart"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	s3sess               *session.Session
	bucket               string
	ServerSideEncryption = "AES256"
)

// SetS3 is used to set the aws session and bucket name for s3 access
func SetS3(accessKey, secretAccessKey, region, bucketName string) error {
	if bucketName == "" {
		return fmt.Errorf("cannot have empty string for s3 bucket name")
	}
	bucket = bucketName
	s3creds := credentials.NewCredentials(&credentials.StaticProvider{Value: credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretAccessKey,
	}})
	var err error
	s3sess, err = session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: s3creds,
	})
	if err != nil {
		err = fmt.Errorf("unable to create aws session for s3: %v", err)
		return err
	}
	return nil
}

// UploadScript is used for uploading test scripts
func UploadScript(testid string, scriptid string, zipFileName string, file multipart.File) error {
	uploadName := "scripts/" + scriptid + "/file/" + zipFileName
	fmt.Println("Uploading", uploadName)
	db.UpdateTestStatus(testid, "Uploading")
	err := uploadObject(file, uploadName)
	if err != nil {
		fmt.Println("Failed to upload", uploadName)
		db.UpdateTestStatus(testid, "Upload Failed")
		return err
	}
	fmt.Println("Finished uploading", uploadName)
	db.UpdateTestStatus(testid, "Ready")
	return err
}

// DownloadScript is used to download test scripts
func DownloadScript(scriptid string, zipFileName string) (*aws.WriteAtBuffer, error) {
	downloadName := "scripts/" + scriptid + "/file/" + zipFileName
	fmt.Println("Downloading", downloadName)
	buff, err := downloadObjectBuffer(downloadName)
	if err != nil {
		fmt.Println("Failed to download", downloadName)
		return buff, err
	}
	fmt.Println("Finished uploading", downloadName)
	return buff, err
}

// UploadAttachment is used for uploading attachments for a test.
func UploadAttachment(testid string, attachmentName string, file multipart.File) error {
	uploadName := "attachments/" + testid + "/file/" + attachmentName
	fmt.Println("Uploading", uploadName)
	err := uploadObject(file, uploadName)
	if err != nil {
		fmt.Println("Failed to upload", uploadName)
		return err
	}
	fmt.Println("Finished uploading", uploadName)
	return err
}

// DownloadAttachment is used for downloading attachments that were uploaded for a test.
func DownloadAttachment(testid string, attachmentName string) (*aws.WriteAtBuffer, error) {
	downloadName := "attachments/" + testid + "/file/" + attachmentName
	fmt.Println("Downloading", downloadName)
	buff, err := downloadObjectBuffer(downloadName)
	if err != nil {
		fmt.Println("Failed to download", downloadName)
		return buff, err
	}
	fmt.Println("Finished uploading", downloadName)
	return buff, err
}

// DeleteAttachment is used for deleting attachments that were uploaded for a test.
func DeleteAttachment(testid string, attachmentName string) error {
	s3Filename := "attachments/" + testid + "/file/" + attachmentName
	err := deleteObject(s3sess, s3Filename)
	if err != nil {
		fmt.Println("Failed to delete object", s3Filename)
		return err
	}

	return nil
}

func uploadObject(file multipart.File, uploadName string) error {
	defer file.Close()

	uploader := s3manager.NewUploader(s3sess)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:               aws.String(bucket),
		ServerSideEncryption: aws.String(ServerSideEncryption),

		// Can also use the `filepath` standard library package to modify the
		// filename as need for an S3 object key. Such as turning absolute path
		// to a relative path.
		Key: aws.String(uploadName),

		// The file to be uploaded. io.ReadSeeker is preferred as the Uploader
		// will be able to optimize memory when uploading large content. io.Reader
		// is supported, but will require buffering of the reader's bytes for
		// each part.
		Body: file,
	})
	if err != nil {
		// Print the error and exit.
		fmt.Println("Unable to upload", err)
		return err
	}

	fmt.Printf("Successfully uploaded %q to %q\n", uploadName, bucket)
	return err
}

func listObjectsV2(sess *session.Session) {
	svc := s3.New(sess)
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int64(100),
	})
	if err != nil {
		fmt.Println("Unable to list items in bucket", err)
	}

	if resp.IsTruncated != nil {
		fmt.Println("Truncation: ", *resp.IsTruncated)
	}

	for _, item := range resp.Contents {
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
		downloadObjectBuffer(*item.Key)
	}

}

func deleteObject(sess *session.Session, fileNameToDelete string) error {
	svc := s3.New(sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileNameToDelete),
	})
	if err != nil {
		fmt.Println("Unable to delete file", err)
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileNameToDelete),
	})
	if err != nil {
		fmt.Println("Error occurred while waiting for object to be deleted", err)
		return err
	}

	fmt.Printf("Object %q successfully deleted\n", fileNameToDelete)
	return nil
}

func downloadObjectBuffer(fileNameToDownload string) (*aws.WriteAtBuffer, error) {
	buff := &aws.WriteAtBuffer{}

	downloader := s3manager.NewDownloader(s3sess)

	_, err := downloader.Download(buff,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(fileNameToDownload),
		})
	if err != nil {
		fmt.Println("Failed to download s3 object into buffer")
		return buff, err
	}

	fmt.Println("Put downloaded object into a buffer")

	return buff, err
}
