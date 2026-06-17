package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OSSStorage struct {
	endpoint     string
	accessKey    string
	secretKey    string
	bucket       string
	customDomain string
	useSSL       bool
	client       *http.Client
}

type OSSConfig struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	CustomDomain string
	UseSSL       bool
}

func NewOSSStorage(cfg OSSConfig) *OSSStorage {
	endpoint := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(cfg.Endpoint), "https://"), "http://")
	endpoint = strings.TrimRight(endpoint, "/")
	return &OSSStorage{endpoint: endpoint, accessKey: cfg.AccessKey, secretKey: cfg.SecretKey, bucket: cfg.Bucket, customDomain: strings.TrimRight(cfg.CustomDomain, "/"), useSSL: cfg.UseSSL, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *OSSStorage) scheme() string {
	if s.useSSL {
		return "https"
	}
	return "http"
}

func (s *OSSStorage) objectURL(key string) string {
	key = strings.TrimLeft(key, "/")
	return fmt.Sprintf("%s://%s.%s/%s", s.scheme(), s.bucket, s.endpoint, key)
}

func (s *OSSStorage) Save(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	key := strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("x-oss-object-acl", "public-read")
	canonical := strings.Join([]string{http.MethodPut, "", contentType, date, "x-oss-object-acl:public-read", "/" + s.bucket + "/" + key}, "\n")
	sig := hmacSHA1Base64([]byte(s.secretKey), canonical)
	req.Header.Set("Authorization", "OSS "+s.accessKey+":"+sig)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oss put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oss put %d: %s", resp.StatusCode, string(body))
	}
	return s.PublicURL(key), nil
}

func (s *OSSStorage) Delete(ctx context.Context, path string) error {
	key := strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	canonical := strings.Join([]string{http.MethodDelete, "", "", date, "/" + s.bucket + "/" + key}, "\n")
	req.Header.Set("Authorization", "OSS "+s.accessKey+":"+hmacSHA1Base64([]byte(s.secretKey), canonical))
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("oss delete %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *OSSStorage) Exists(ctx context.Context, path string) bool {
	key := strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.objectURL(key), nil)
	if err != nil {
		return false
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	canonical := strings.Join([]string{http.MethodHead, "", "", date, "/" + s.bucket + "/" + key}, "\n")
	req.Header.Set("Authorization", "OSS "+s.accessKey+":"+hmacSHA1Base64([]byte(s.secretKey), canonical))
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (s *OSSStorage) PublicURL(path string) string {
	key := strings.TrimLeft(path, "/")
	if s.customDomain != "" {
		return s.customDomain + "/" + key
	}
	return s.objectURL(key)
}

func hmacSHA1Base64(key []byte, data string) string {
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
