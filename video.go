package main

import (
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type Video struct {
	pipe io.ReadCloser
}

// Returns a video object
func getVideo(path string) (Video, error) {
	cmd := exec.Command("ffmpeg", "-i", path, "-pix_fmt", "rgb48le", "-c:v",
		"rawvideo", "-f", "rawvideo", "-")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return Video{}, err
	}
	err = cmd.Start()
	if err != nil {
		return Video{}, err
	}

	return Video{pipe}, err
}

// Returns the frame in raw bytes
func (v *Video) getFrame() (*[]byte, error) {
	bytes := width * height * 3 * 2
	buf := make([]byte, bytes)
	n, err := io.ReadFull(v.pipe, buf)
	if err != nil {
		return nil, err
	}
	if n != bytes {
		return nil, errors.New("not enough bytes")
	}

	return &buf, nil
}

// width, height, error
func getVideoResolution(path string) (int, int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	strout := string(out)
	strout = strings.Replace(strout, "\n", "", -1)
	split := strings.Split(strout, "x")
	if len(split) != 2 {
		return 0, 0, errors.New("error when getting resolution")
	}
	width, err := strconv.Atoi(split[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(split[1])
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}
