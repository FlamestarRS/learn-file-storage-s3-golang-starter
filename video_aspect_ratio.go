package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
)

func (cfg *apiConfig) getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal("error running ffprobe")
	}

	type ffprobeOutput struct {
		Streams []struct {
			Height int `json:"height"`
			Width  int `json:"width"`
		}
	}

	var dimensions ffprobeOutput
	err = json.Unmarshal(out.Bytes(), &dimensions)
	if err != nil {
		log.Fatal("error unmarhsalling ffprobe")
	}

	aspectRatio := ""
	if (dimensions.Streams[0].Width / 16) == (dimensions.Streams[0].Height / 9) {
		aspectRatio = "landscape"
	} else if (dimensions.Streams[0].Width / 9) == (dimensions.Streams[0].Height / 16) {
		aspectRatio = "portrait"
	} else {
		aspectRatio = "other"
	}

	return aspectRatio, nil

}
