package downloader

import (
	"context"
	"errors"
	"log"
	"testing"
)

type mockRemoteCache struct {
	store map[string][]string
}

func (m *mockRemoteCache) Put(ctx context.Context, key, urlStr string) error {
	if m.store == nil {
		m.store = make(map[string][]string)
	}
	m.store[key] = append(m.store[key], urlStr)
	return nil
}

func (m *mockRemoteCache) Get(ctx context.Context, key string) ([]string, error) {
	return m.store[key], nil
}

func TestRemoteURLCacheDisabled(t *testing.T) {
	d := NewDownloader()
	ctx := context.Background()
	if err := d.PutRemoteURL(ctx, "testKey", "https://example.com"); !errors.Is(err, ErrRemoteURLCacheDisabled) {
		t.Fatalf("expected disabled cache error, got %v", err)
	}
	if _, err := d.GetRemoteURLs(ctx, "testKey"); !errors.Is(err, ErrRemoteURLCacheDisabled) {
		t.Fatalf("expected disabled cache error, got %v", err)
	}
}

func TestRemoteURLCacheHook(t *testing.T) {
	cache := &mockRemoteCache{}
	d := NewDownloader(WithRemoteURLCache(cache))
	ctx := context.Background()
	if err := d.PutRemoteURL(ctx, "testKey", "https://example.com"); err != nil {
		t.Fatalf("unexpected error putting remote URL: %v", err)
	}
	urls, err := d.GetRemoteURLs(ctx, "testKey")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || urls[0] != "https://example.com" {
		t.Fatalf("unexpected urls: %v", urls)
	}
}

func TestRedirectRange(t *testing.T) {
	d := NewDownloader()
	virtioURL1 := "https://dl.istoreos.com/iStoreOS/Virtual/virtio-win-0.1.271.iso"
	virtioURL2 := "https://fw0.koolcenter.com/iStoreOS/Virtual/virtio-win-0.1.271.iso"
	//virtioURL := "https://fw21.koolcenter.com:60010/iStoreOS/Virtual/virtio-win-0.1.271.iso"
	for _, loc := range []string{virtioURL1, virtioURL2} {
		total, remoteModTime, err := d.HeadInfo(loc)
		if err != nil {
			t.Fatal(err)
		}
		if total < 4096 {
			t.Fatalf("Unexpected total size: %d", total)
		}
		log.Println("total=", total, "remoteModTime=", remoteModTime)
	}
}
