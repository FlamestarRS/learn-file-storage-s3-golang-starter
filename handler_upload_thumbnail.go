package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handlerUploadThumbnail called")
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't retrieve image data", err)
		return
	}
	defer file.Close()

	mediaHeader := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(mediaHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "missing Content-Type for thumbnail", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "file must be jpeg or png", nil)
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

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		panic("failed to generate random bytes")
	}
	encoded := base64.RawURLEncoding.EncodeToString(key)
	dataPath := getAssetPath([]byte(encoded), mediaType)
	dataDiskPath := cfg.getAssetDiskPath(dataPath)

	url := cfg.getAssetURL(dataPath)
	video.ThumbnailURL = &url

	thumbnailFile, err := os.Create(dataDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't create file", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = io.Copy(thumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't copy data from file", err)
		return
	}

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
