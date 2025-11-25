package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/linkease/fastpve/downloader"
	"github.com/linkease/fastpve/utils"
)

func downloadURL(ctx context.Context, downer *downloader.Downloader, statusPath string,
	status *downloader.DownloadStatus) error {
	progressCh := make(chan *downloader.ProgressInfo, 8)
	go func() {
		for progress := range progressCh {
			downloader.UpdateDownloadStatus(progress.Status, statusPath)
			log.Println("speed=", utils.ByteCountDecimal(uint64(progress.Speed)), "progress=", progress.Progress)
		}
	}()
	err := downer.ResumableDownloader(ctx, status.Url, status.TargetFile, status, progressCh)
	close(progressCh)
	if err == nil {
		time.Sleep(time.Second)
		os.Remove(statusPath)
	}
	return err
}
