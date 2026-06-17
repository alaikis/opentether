package storage

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestS3SignRequestIncludesACLHeader(t *testing.T) {
	s := NewS3Storage(S3Config{Endpoint: "oss-cn-hongkong.aliyuncs.com", Region: "cn-hongkong", AccessKey: "ak", SecretKey: "sk", Bucket: "e2c", UseSSL: true})
	req, err := http.NewRequest(http.MethodPut, s.objectURL("a.txt"), bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-amz-acl", "public-read")
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	if err := s.signRequest(req, "UNSIGNED-PAYLOAD"); err != nil {
		t.Fatal(err)
	}
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "x-amz-acl") {
		t.Fatalf("expected signed headers to include x-amz-acl, got %s", auth)
	}
}
