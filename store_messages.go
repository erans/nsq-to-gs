package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	storage "google.golang.org/api/storage/v1"
)

// PrintMessages prints messages to the screen:
func PrintMessages(fileData []byte) error {

	var now = time.Now().UTC()
	fileName := fmt.Sprintf("%s-%d%02d%02d_%02d%02d.%s.gz", *gsFilePrefix, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), *gsFileExtension)
	fullFilePath := fmt.Sprintf("%s/%d/%02d/%02d/%s", *gsPath, now.Year(), int(now.Month()), now.Day(), fileName)

	log.Infof("Would store in '%v'", fullFilePath)

	log.Debugf("Messages: %v", string(fileData))

	return nil
}

// StoreMessages store messages to GS:
func StoreMessages(fileData []byte) error {

	// Something to compress the fileData into:
	var fileDataBytes bytes.Buffer
	gzFileData := gzip.NewWriter(&fileDataBytes)
	gzFileData.Write(fileData)
	gzFileData.Close()

	log.Infof("Storing %d bytes...", len(fileDataBytes.Bytes()))

	// Build the filename we'll use for GS:
	var now = time.Now().UTC()
	fileName := fmt.Sprintf("%s-%d%02d%02d_%02d%02d.%s.gz", *gsFilePrefix, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), *gsFileExtension)

	fullFilePath := fmt.Sprintf("%s/%d/%02d/%02d/%s", *gsPath, now.Year(), int(now.Month()), now.Day(), fileName)

	// Authentication is provided by the gcloud tool when running locally, and
	// by the associated service account when running on Compute Engine.
	client, err := google.DefaultClient(context.Background(), storage.DevstorageFullControlScope)
	if err != nil {
		log.Criticalf("Unable to get default client: %v", err)
	}
	service, err := storage.New(client)
	if err != nil {
		log.Criticalf("Unable to create storage service: %v", err)
	}

	// Insert an object into a bucket.
	object := &storage.Object{Name: fullFilePath}
	if res, err := service.Objects.Insert(*gsBucket, object).Media(bytes.NewReader(fileDataBytes.Bytes())).Do(); err == nil {
		log.Infof("Created object %v at location %v\n\n", res.Name, res.SelfLink)
		log.Infof("Stored file (%v) on GS", fullFilePath)
	} else {
		log.Criticalf("Failed to put file (%v) on GS (%v)", fullFilePath, err)
		os.Exit(2)
	}

	return nil
}
