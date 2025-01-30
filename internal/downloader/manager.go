package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"
)

type Manager struct {
	client *http.Client
}

func NewManager() *Manager {
	return &Manager{
		client: &http.Client{
			Timeout: 0,
		},
	}
}

type DownloadResult struct {
	Path string
	Size int64
}

func (m *Manager) Download(url, destDir string) (*DownloadResult, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	resp, err := m.client.Head(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := resp.ContentLength

	resp, err = m.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	fileName := filepath.Base(url)
	destPath := filepath.Join(destDir, fileName)
	file, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	bar := progressbar.NewOptions64(
		fileSize,
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionUseIECUnits(false),
	)

	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	return &DownloadResult{
		Path: destPath,
		Size: fileSize,
	}, nil
}
