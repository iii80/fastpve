package main

import (
	"context"
	"fmt"
	"testing"
)

func TestDownloadVirtIO(t *testing.T) {
	downer := newDownloader()
	ctx := context.TODO()
	isoPath := "./"
	virtPath := "./status.ops"
	realPath, err := downloadVirtIO(ctx, downer, isoPath, virtPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("realPath=", realPath)
}
