package embedding

import "testing"

func TestTFIDFEmbedderProducesNonZeroVectors(t *testing.T) {
	config := map[string]interface{}{
		"corpus": []string{"hello world", "foo bar"},
	}
	e, err := Create("tfidf", config)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	vec, err := e.Embed("hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vec) == 0 {
		t.Fatal("expected non-zero vector")
	}
	hasNonZero := false
	for _, v := range vec {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Fatal("expected vector to have non-zero values")
	}
}

func TestTFIDFDims(t *testing.T) {
	config := map[string]interface{}{
		"corpus": []string{"hello world", "foo bar"},
	}
	e, err := Create("tfidf", config)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.Dims() != 4 {
		t.Fatalf("expected dims 4, got %d", e.Dims())
	}
}

func TestONNXEmbedderRegistersCorrectly(t *testing.T) {
	e, err := Create("onnx", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.Name() != "onnx" {
		t.Fatalf("expected name onnx, got %s", e.Name())
	}
	if e.Dims() != 384 {
		t.Fatalf("expected dims 384, got %d", e.Dims())
	}
}

func TestCreateReturnsDefaultWhenNoProvider(t *testing.T) {
	e, err := Create("", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.Name() != "tfidf" {
		t.Fatalf("expected default tfidf, got %s", e.Name())
	}
}

func TestCreateFallsBackToTFIDFForUnknownProvider(t *testing.T) {
	e, err := Create("unknown-provider", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.Name() != "tfidf" {
		t.Fatalf("expected fallback tfidf, got %s", e.Name())
	}
}
