package vectorstore

import (
	"testing"
)

func TestMemoryStoreUpdate(t *testing.T) {
	store, err := CreateStore("memory", nil)
	if err != nil {
		t.Fatalf("CreateStore failed: %v", err)
	}
	if err := store.Index("s1", "skill1", []float64{1, 0}); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	if err := store.Update("s1", "skill1-updated", []float64{0, 1}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if store.Count() != 1 {
		t.Fatalf("expected count 1, got %d", store.Count())
	}
	results, err := store.Search([]float64{0, 1}, 1, 0.0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if results[0].SkillName != "skill1-updated" {
		t.Fatalf("expected skill1-updated, got %s", results[0].SkillName)
	}
}

func TestMemoryStoreHybridSearch(t *testing.T) {
	store, err := CreateStore("memory", nil)
	if err != nil {
		t.Fatalf("CreateStore failed: %v", err)
	}
	if err := store.Index("s1", "skill1", []float64{1, 1}); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	if err := store.Index("s2", "skill2", []float64{0, 1}); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	results, err := store.HybridSearch([]float64{1, 0}, 2, 0.0, 0.5)
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected non-empty results")
	}
}

func TestMemoryStoreSearchWithMetadata(t *testing.T) {
	store, err := CreateStore("memory", nil)
	if err != nil {
		t.Fatalf("CreateStore failed: %v", err)
	}
	mem, ok := store.(*MemoryStore)
	if !ok {
		t.Fatal("expected *MemoryStore")
	}
	mem.vectors["s1"] = vectorEntry{skillID: "s1", skillName: "skill1", vector: []float64{1, 0}, metadata: map[string]string{"type": "a"}}
	mem.vectors["s2"] = vectorEntry{skillID: "s2", skillName: "skill2", vector: []float64{0, 1}, metadata: map[string]string{"type": "b"}}
	results, err := store.SearchWithMetadata([]float64{1, 0}, 2, 0.0, map[string]string{"type": "a"})
	if err != nil {
		t.Fatalf("SearchWithMetadata failed: %v", err)
	}
	if len(results) != 1 || results[0].SkillID != "s1" {
		t.Fatalf("expected 1 result with s1, got %d results", len(results))
	}
}

func TestBM25Score(t *testing.T) {
	v1 := []float64{1, 0}
	v2 := []float64{1, 0}
	score := BM25Score(v1, v2)
	if score <= 0 {
		t.Fatalf("expected positive score, got %f", score)
	}
}
