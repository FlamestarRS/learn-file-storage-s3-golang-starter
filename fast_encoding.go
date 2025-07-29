package main

import (
	"fmt"
	"os/exec"
)

func (cfg *apiConfig) processVideoForFastStart(filePath string) (string, error) {
	newFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffmpeg fast start")
	}
	return newFilePath, nil
}
