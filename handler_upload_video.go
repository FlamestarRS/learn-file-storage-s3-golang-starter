package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handlerUploadVideo called")
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		if err == http.ErrBodyNotAllowed {
			respondWithError(w, http.StatusRequestEntityTooLarge, "Request entity too large. Maximum allowed size is 1MB.", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error parsing form", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't retrieve video data", err)
		return
	}
	defer file.Close()

	mediaHeader := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(mediaHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "missing Content-Type for video", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "file must be mp4", nil)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't retrieve video metadata", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "not authorized to update video", nil)
		return
	}

	videoFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't create file", err)
		return
	}
	defer os.Remove(videoFile.Name())
	defer videoFile.Close()

	_, err = io.Copy(videoFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't copy data from file", err)
		return
	}

	_, err = videoFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't reset file pointer", err)
		return
	}

	absPath, err := filepath.Abs(videoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't get abs path", err)
		return
	}

	ratio, err := cfg.getVideoAspectRatio(absPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't get aspect ratio", err)
		return
	}
	fastProcessing, err := cfg.processVideoForFastStart(absPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't complete fastprocessing", err)
		return
	}
	processedFile, err := os.Open(fastProcessing)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't open processed file", err)
		return
	}
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		panic("failed to generate random bytes")
	}
	encoded := base64.RawURLEncoding.EncodeToString(key)
	dataPath := getAssetPath([]byte(encoded), mediaType)
	fullKey := ratio + "/" + dataPath

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fullKey),
		Body:        processedFile,
		ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't upload video to aws", err)
		return
	}

	url := cfg.s3Bucket + "," + fullKey
	video.VideoURL = &url

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't update video", err)
		return
	}

	presignedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error presigning video handlerUploadVideo", err)
		return
	}

	respondWithJSON(w, http.StatusOK, presignedVideo)
}
