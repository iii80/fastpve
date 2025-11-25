package main

import "github.com/linkease/fastpve/downloader"

// newDownloader creates a downloader without remote URL cache.
// This is the default for open-source builds.
func newDownloader() *downloader.Downloader {
	return downloader.NewDownloader()
}
