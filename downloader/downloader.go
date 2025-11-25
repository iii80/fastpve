package downloader

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var ErrRemoteURLCacheDisabled = errors.New("remote URL cache is disabled")

type RemoteURLCache interface {
	Put(ctx context.Context, key, urlStr string) error
	Get(ctx context.Context, key string) ([]string, error)
}

var remoteURLCacheProvider func() RemoteURLCache

// RegisterRemoteURLCacheProvider allows optional injection of a remote URL cache.
// It is typically called in init() of a private module that is blank-imported.
func RegisterRemoteURLCacheProvider(fn func() RemoteURLCache) {
	remoteURLCacheProvider = fn
}

type DownloaderOption func(*Downloader)

type Downloader struct {
	client             *http.Client
	noRedirectClient   *http.Client
	remoteURLCache     RemoteURLCache
	remoteCacheEnabled bool
}

type noopRemoteURLCache struct{}

func (noopRemoteURLCache) Put(ctx context.Context, key, urlStr string) error {
	return ErrRemoteURLCacheDisabled
}

func (noopRemoteURLCache) Get(ctx context.Context, key string) ([]string, error) {
	return nil, ErrRemoteURLCacheDisabled
}

func WithRemoteURLCache(cache RemoteURLCache) DownloaderOption {
	return func(d *Downloader) {
		if cache == nil {
			return
		}
		d.remoteURLCache = cache
		d.remoteCacheEnabled = true
	}
}

func NewDownloader(opts ...DownloaderOption) *Downloader {
	d := &Downloader{
		client: &http.Client{},
		noRedirectClient: &http.Client{
			Timeout: time.Second * 10,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		remoteURLCache:     noopRemoteURLCache{},
		remoteCacheEnabled: false,
	}
	if remoteURLCacheProvider != nil {
		if cache := remoteURLCacheProvider(); cache != nil {
			d.remoteURLCache = cache
			d.remoteCacheEnabled = true
		}
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Downloader) DefaultClient() *http.Client {
	return d.client
}

func (d *Downloader) RemoteURLCacheEnabled() bool {
	return d.remoteCacheEnabled
}

func (d *Downloader) PutRemoteURL(ctx context.Context, key, urlStr string) error {
	if !d.remoteCacheEnabled {
		return ErrRemoteURLCacheDisabled
	}
	return d.remoteURLCache.Put(ctx, key, urlStr)
}

func (d *Downloader) GetRemoteURLs(ctx context.Context, key string) ([]string, error) {
	if !d.remoteCacheEnabled {
		return nil, ErrRemoteURLCacheDisabled
	}
	return d.remoteURLCache.Get(ctx, key)
}
