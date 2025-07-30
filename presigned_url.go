package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) generatePresignedURL(bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(cfg.s3Client)
	req, err := client.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, fmt.Errorf("why is there no video url")
	}
	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return video, fmt.Errorf("why arent there two parts? url: %v\nvideoid: %v", parts, video.ID)
	}
	presignedURL, err := cfg.generatePresignedURL(parts[0], parts[1], time.Hour)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL

	return video, nil
}
