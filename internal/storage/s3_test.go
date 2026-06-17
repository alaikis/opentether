package storage

import "testing"

func TestS3VirtualHostedStyleURL(t *testing.T) {
	s := NewS3Storage(S3Config{Endpoint: "oss-cn-hongkong.aliyuncs.com", Bucket: "mybucket", UseSSL: true, PathStyle: false})
	got := s.objectURL("reports/a.pdf")
	want := "https://mybucket.oss-cn-hongkong.aliyuncs.com/reports/a.pdf"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestS3PathStyleURL(t *testing.T) {
	s := NewS3Storage(S3Config{Endpoint: "localhost:9000", Bucket: "mybucket", UseSSL: false, PathStyle: true})
	got := s.objectURL("reports/a.pdf")
	want := "http://localhost:9000/mybucket/reports/a.pdf"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestS3CustomDomainPublicURL(t *testing.T) {
	s := NewS3Storage(S3Config{Endpoint: "oss-cn-hongkong.aliyuncs.com", Bucket: "mybucket", CustomDomain: "https://cdn.example.com/", UseSSL: true})
	got := s.PublicURL("reports/a.pdf")
	want := "https://cdn.example.com/reports/a.pdf"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
