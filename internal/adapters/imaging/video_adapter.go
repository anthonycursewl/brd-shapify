package imaging

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type VideoProcessorAdapter struct {
	ffmpegPath string
	workDir    string
}

func NewVideoProcessorAdapter() (*VideoProcessorAdapter, error) {
	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	workDir, err := os.MkdirTemp("", "video-processing")
	if err != nil {
		return nil, err
	}

	return &VideoProcessorAdapter{
		ffmpegPath: ffmpegPath,
		workDir:    workDir,
	}, nil
}

func (v *VideoProcessorAdapter) ExtractThumbnail(videoData []byte, timestamp string) ([]byte, error) {
	if timestamp == "" {
		timestamp = "00:00:01"
	}

	inputPath := filepath.Join(v.workDir, "input_"+fmt.Sprint(time.Now().UnixNano()))
	outputPath := filepath.Join(v.workDir, "thumb_"+fmt.Sprint(time.Now().UnixNano())+".jpg")

	if err := os.WriteFile(inputPath, videoData, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)

	cmd := exec.Command(v.ffmpegPath, "-y", "-i", inputPath, "-ss", timestamp, "-vframes", "1", "-q:v", "2", outputPath)
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("thumbnail extraction failed: %w", err)
	}
	defer os.Remove(outputPath)

	return os.ReadFile(outputPath)
}

func (v *VideoProcessorAdapter) ConvertVideo(videoData []byte, format string, quality int) ([]byte, error) {
	if quality <= 0 {
		quality = 23
	}
	if quality > 51 {
		quality = 51
	}

	inputPath := filepath.Join(v.workDir, "input_"+fmt.Sprint(time.Now().UnixNano()))
	ext := "mp4"
	if format != "" {
		ext = format
	}
	outputPath := fmt.Sprintf("output_%d.%s", time.Now().UnixNano(), ext)

	if err := os.WriteFile(inputPath, videoData, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)

	var crf string
	switch ext {
	case "webm", "mkv":
		crf = fmt.Sprintf("-crf %d -b:v 0", quality)
	default:
		crf = fmt.Sprintf("-crf %d", quality)
	}

	filterFlag := "-c:v libx264"
	if ext == "webm" {
		filterFlag = "-c:v libvpx-vp9"
	}

	cmdStr := fmt.Sprintf("%s -y -i %s %s %s %s", v.ffmpegPath, inputPath, filterFlag, crf, outputPath)
	cmd := exec.Command("sh", "-c", cmdStr)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("video conversion failed: %w", err)
	}
	defer os.Remove(outputPath)

	return os.ReadFile(outputPath)
}

func (v *VideoProcessorAdapter) ExtractAudio(videoData []byte, format string) ([]byte, error) {
	if format == "" {
		format = "mp3"
	}

	inputPath := filepath.Join(v.workDir, "input_"+fmt.Sprint(time.Now().UnixNano()))
	outputPath := filepath.Join(v.workDir, "audio_"+fmt.Sprint(time.Now().UnixNano())+"."+format)

	if err := os.WriteFile(inputPath, videoData, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)

	var codec string
	switch format {
	case "mp3":
		codec = "libmp3lame"
	case "aac":
		codec = "aac"
	case "ogg":
		codec = "libvorbis"
	default:
		codec = "libmp3lame"
	}

	cmd := exec.Command(v.ffmpegPath, "-y", "-i", inputPath, "-vn", "-acodec", codec, outputPath)
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("audio extraction failed: %w", err)
	}
	defer os.Remove(outputPath)

	return os.ReadFile(outputPath)
}

func (v *VideoProcessorAdapter) TrimVideo(videoData []byte, start, duration string) ([]byte, error) {
	if start == "" {
		start = "00:00:00"
	}
	if duration == "" {
		duration = "10"
	}

	inputPath := filepath.Join(v.workDir, "input_"+fmt.Sprint(time.Now().UnixNano()))
	outputPath := filepath.Join(v.workDir, "trim_"+fmt.Sprint(time.Now().UnixNano())+".mp4")

	if err := os.WriteFile(inputPath, videoData, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)

	cmd := exec.Command(v.ffmpegPath, "-y", "-i", inputPath, "-ss", start, "-t", duration, "-c", "copy", outputPath)
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("video trim failed: %w", err)
	}
	defer os.Remove(outputPath)

	return os.ReadFile(outputPath)
}

func (v *VideoProcessorAdapter) Close() error {
	os.RemoveAll(v.workDir)
	return nil
}

func (v *VideoProcessorAdapter) ProcessAsync(jobID string, videoData []byte, operation string, params map[string]string, webhookURL string) (string, error) {
	var result []byte
	var err error

	switch operation {
	case "convert":
		result, err = v.ConvertVideo(videoData, params["format"], 0)
	case "thumbnail":
		result, err = v.ExtractThumbnail(videoData, params["timestamp"])
	case "audio":
		result, err = v.ExtractAudio(videoData, params["format"])
	case "trim":
		result, err = v.TrimVideo(videoData, params["start"], params["duration"])
	default:
		return "", fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return "", err
	}

	if webhookURL != "" {
		go func() {
			client := &http.Client{Timeout: 10 * time.Second}
			client.Post(webhookURL, "application/json", bytes.NewReader(result))
		}()
	}

	return fmt.Sprintf("%d_%s.%s", time.Now().UnixNano(), jobID, params["format"]), nil
}
