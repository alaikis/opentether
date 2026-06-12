package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// S3Storage stores files on any S3-compatible object storage.
//
// Supported services:
//   - AWS S3               (endpoint="s3.<region>.amazonaws.com", region="us-east-1", pathStyle=false)
//   - Alibaba OSS           (endpoint="oss-<region>.aliyuncs.com", region="oss-<region>", pathStyle=false)
//   - Tencent COS           (endpoint="cos.<region>.myqcloud.com", region="<region>", pathStyle=false)
//   - MinIO                 (endpoint="localhost:9000", pathStyle=true)
//   - Huawei OBS            (endpoint="obs.<region>.myhuaweicloud.com", pathStyle=false)
//   - Any S3-compatible     (set endpoint + region + pathStyle accordingly)
type S3Storage struct {
	endpoint  string // host[:port]
	region    string
	accessKey string
	secretKey string
	bucket    string
	useSSL    bool
	pathStyle bool // true for MinIO-style, false for virtual-hosted
	client    *http.Client
}

// S3Config holds S3-compatible storage configuration.
type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PathStyle bool
}

func NewS3Storage(cfg S3Config) *S3Storage {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	return &S3Storage{
		endpoint:  cfg.Endpoint,
		region:    cfg.Region,
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		bucket:    cfg.Bucket,
		useSSL:    cfg.UseSSL,
		pathStyle: cfg.PathStyle,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *S3Storage) baseURL() string {
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return scheme + "://" + s.endpoint
}

func (s *S3Storage) objectURL(key string) string {
	if s.pathStyle {
		return s.baseURL() + "/" + s.bucket + "/" + key
	}
	return s.baseURL() + "/" + key
}

func (s *S3Storage) bucketURL() string {
	if s.pathStyle {
		return s.baseURL() + "/" + s.bucket
	}
	return s.baseURL()
}

func (s *S3Storage) Save(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	body := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, "PUT", s.objectURL(path), body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("x-amz-acl", "public-read")

	// Compute payload hash
	h := sha256.New()
	h.Write(data)
	payloadHash := hex.EncodeToString(h.Sum(nil))
	req.Header.Set("x-amz-content-sha256", payloadHash)

	if err := s.signRequest(req, payloadHash); err != nil {
		return "", fmt.Errorf("sign request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("s3 put: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 put %d: %s", resp.StatusCode, string(respBody))
	}

	return s.PublicURL(path), nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.objectURL(path), nil)
	if err != nil {
		return err
	}
	payloadHash := "UNSIGNED-PAYLOAD"
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if err := s.signRequest(req, payloadHash); err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *S3Storage) Exists(ctx context.Context, path string) bool {
	req, err := http.NewRequestWithContext(ctx, "HEAD", s.objectURL(path), nil)
	if err != nil {
		return false
	}
	payloadHash := "UNSIGNED-PAYLOAD"
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if err := s.signRequest(req, payloadHash); err != nil {
		return false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (s *S3Storage) PublicURL(path string) string {
	return s.objectURL(path)
}

// ── AWS Signature V4 (stdlib only) ────────────────────────────────────

func (s *S3Storage) signRequest(req *http.Request, payloadHash string) error {
	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	datetimeStr := now.Format("20060102T150405Z")
	service := "s3"

	// Canonical request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQuery := req.URL.RawQuery

	canonicalHeaders := "host:" + req.Host + "\n"
	canonicalHeaders += "x-amz-content-sha256:" + payloadHash + "\n"
	canonicalHeaders += "x-amz-date:" + datetimeStr + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalReq := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := dateStr + "/" + s.region + "/" + service + "/aws4_request"
	canonicalReqHash := sha256Hex(canonicalReq)
	stringToSign := strings.Join([]string{
		algorithm,
		datetimeStr,
		credentialScope,
		canonicalReqHash,
	}, "\n")

	// Signing key
	signingKey := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256(
		[]byte("AWS4"+s.secretKey),
		dateStr),
		s.region),
		service),
		"aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	// Authorization header
	auth := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, s.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-amz-date", datetimeStr)

	return nil
}

func sha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// ── Factory ───────────────────────────────────────────────────────────

// New creates a Driver from config.
func New(cfg Config) (Driver, error) {
	switch cfg.Type {
	case "local":
		return NewLocalStorage(cfg.Local.Path, cfg.Local.BaseURL)
	case "s3":
		s3 := NewS3Storage(S3Config{
			Endpoint:  cfg.S3.Endpoint,
			Region:    cfg.S3.Region,
			AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey,
			Bucket:    cfg.S3.Bucket,
			UseSSL:    cfg.S3.UseSSL,
			PathStyle: cfg.S3.PathStyle,
		})
		return s3, nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// Config holds storage configuration.
type Config struct {
	Type  string      // "local" or "s3"
	Local LocalConfig `yaml:"local"`
	S3    S3ConfigRaw `yaml:"s3"`
}

type LocalConfig struct {
	Path    string `yaml:"path"`
	BaseURL string `yaml:"base_url"`
}

type S3ConfigRaw struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
	PathStyle bool   `yaml:"path_style"`
}
