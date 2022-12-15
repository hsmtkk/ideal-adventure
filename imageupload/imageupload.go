package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
)

func main() {
	command := &cobra.Command{
		Args: cobra.ExactArgs(2),
		Run:  run,
		Use:  "imageupload bucket list.csv",
	}
	if err := command.Execute(); err != nil {
		log.Fatal(err)
	}
}

const (
	applesPath   = "../image/train/apples"
	tomatoesPath = "../image/train/tomatoes"
)

func run(cmd *cobra.Command, args []string) {
	bucketName := args[0]
	listCSV := args[1]

	outFile, err := os.Create(listCSV)
	if err != nil {
		log.Fatalf("os.Create failed; %v", err.Error())
	}
	defer outFile.Close()
	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	ctx := context.Background()

	clt, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient failed; %v", err.Error())
	}
	defer clt.Close()
	handle := clt.Bucket(bucketName)

	apples, err := os.ReadDir(applesPath)
	if err != nil {
		log.Fatalf("os.ReadDir failed; %v", err.Error())
	}
	for _, entry := range apples {
		fileName := entry.Name()
		log.Printf("uploading %s", fileName)
		srcPath := filepath.Join(applesPath, fileName)
		dstKey := "apples/" + fileName
		if err := upload(ctx, handle, srcPath, dstKey); err != nil {
			log.Fatalf("upload failed; %v", err.Error())
		}
		url := fmt.Sprintf("gs://%s/%s", bucketName, dstKey)
		if err := writer.Write([]string{url, "apple"}); err != nil {
			log.Fatalf("csv.Writer.Write failed; %v", err.Error())
		}
	}

	tomatoes, err := os.ReadDir(tomatoesPath)
	if err != nil {
		log.Fatalf("os.ReadDir failed; %v", err.Error())
	}
	for _, entry := range tomatoes {
		fileName := entry.Name()
		log.Printf("uploading %s", fileName)
		srcPath := filepath.Join(tomatoesPath, fileName)
		dstKey := "tomatoes/" + fileName
		if err := upload(ctx, handle, srcPath, dstKey); err != nil {
			log.Fatalf("upload failed; %v", err.Error())
		}
		url := fmt.Sprintf("gs://%s/%s", bucketName, dstKey)
		if err := writer.Write([]string{url, "tomato"}); err != nil {
			log.Fatalf("csv.Writer.Write failed; %v", err.Error())
		}
	}
}

func upload(ctx context.Context, handle *storage.BucketHandle, srcPath, dstKey string) error {
	reader, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("os.Open failed; %w", err)
	}
	defer reader.Close()
	writer := handle.Object(dstKey).NewWriter(ctx)
	defer writer.Close()
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("io.Copy failed; %w", err)
	}
	return nil
}
