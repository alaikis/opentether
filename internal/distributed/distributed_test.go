package distributed

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryGossipProtocol(t *testing.T) {
	gossip := NewInMemoryGossipProtocol()

	err := gossip.Join(context.Background(), []string{"node1", "node2"})
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	peers, err := gossip.Peers()
	if err != nil {
		t.Fatalf("Peers failed: %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(peers))
	}

	msg := &GossipMessage{
		Type:     "ping",
		FromNode: "node1",
	}
	if err := gossip.Broadcast(msg); err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	if err := gossip.Leave(context.Background()); err != nil {
		t.Fatalf("Leave failed: %v", err)
	}
}

func TestInMemoryGossipProtocolOnMessage(t *testing.T) {
	gossip := NewInMemoryGossipProtocol()

	received := make(chan *GossipMessage, 1)
	gossip.OnMessage(func(m *GossipMessage) {
		received <- m
	})

	msg := &GossipMessage{
		Type:     "hello",
		FromNode: "node1",
	}
	if err := gossip.Broadcast(msg); err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	select {
	case m := <-received:
		if m.Type != "hello" {
			t.Fatalf("Expected message type hello, got %s", m.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestInMemoryTaskDistributor(t *testing.T) {
	distributor := NewInMemoryTaskDistributor()

	assignment := &TaskAssignment{
		TaskID: "task_1",
		NodeID: "node_1",
	}
	if err := distributor.SubmitTask(context.Background(), assignment); err != nil {
		t.Fatalf("SubmitTask failed: %v", err)
	}

	results, err := distributor.TaskResults(context.Background(), "task_1")
	if err != nil {
		t.Fatalf("TaskResults failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results initially, got %d", len(results))
	}

	time.Sleep(200 * time.Millisecond)

	results, err = distributor.TaskResults(context.Background(), "task_1")
	if err != nil {
		t.Fatalf("TaskResults failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result after execution, got %d", len(results))
	}
	if results[0].Status != "success" {
		t.Fatalf("Expected success status, got %s", results[0].Status)
	}

	if err := distributor.CancelTask(context.Background(), "task_1"); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
}

func TestInMemoryFailureDetector(t *testing.T) {
	detector := NewInMemoryFailureDetector()

	if err := detector.RegisterHeartbeat("node1"); err != nil {
		t.Fatalf("RegisterHeartbeat failed: %v", err)
	}
	if !detector.IsAlive("node1") {
		t.Fatal("node1 should be alive after heartbeat")
	}

	if err := detector.RecordFailure("node1"); err != nil {
		t.Fatalf("RecordFailure failed: %v", err)
	}

	failed, err := detector.DetectFailures(time.Now().Add(31 * time.Second))
	if err != nil {
		t.Fatalf("DetectFailures failed: %v", err)
	}
	if len(failed) != 1 || failed[0] != "node1" {
		t.Fatalf("Expected node1 to be detected as failed, got %v", failed)
	}

	if detector.IsAlive("nonexistent") {
		t.Fatal("nonexistent should not be alive (no heartbeat)")
	}
}

func TestDistributedCache(t *testing.T) {
	gossip := NewInMemoryGossipProtocol()
	cache := NewDistributedCache(gossip, "node1", 5*time.Minute)

	cache.Set("key1", "value1")
	val, ok := cache.Get("key1")
	if !ok || val != "value1" {
		t.Fatalf("Expected key1=value1, got %v, ok=%v", val, ok)
	}

	cache.Delete("key1")
	_, ok = cache.Get("key1")
	if ok {
		t.Fatal("Expected key1 to be deleted")
	}

	cache.Set("key2", 42)
	val, ok = cache.Get("key2")
	if !ok || val != 42 {
		t.Fatalf("Expected key2=42, got %v, ok=%v", val, ok)
	}

	if err := cache.Sync(context.Background()); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := cache.MarshalEntries()
	if err != nil {
		t.Fatalf("MarshalEntries failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("MarshalEntries should return non-empty data")
	}
}

func TestDistributedCacheExpiry(t *testing.T) {
	gossip := NewInMemoryGossipProtocol()
	cache := NewDistributedCache(gossip, "node1", 100*time.Millisecond)

	cache.Set("key1", "value1")
	val, ok := cache.Get("key1")
	if !ok || val != "value1" {
		t.Fatalf("Expected key1=value1 initially, got %v, ok=%v", val, ok)
	}

	time.Sleep(200 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Fatal("Expected key1 to be expired")
	}
}
