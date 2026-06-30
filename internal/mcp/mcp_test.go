package mcp

import (
	"context"
	"testing"
)

func TestInMemoryServerRegistry(t *testing.T) {
	registry := NewInMemoryServerRegistry()

	config := &MCPServerConfig{
		ID:        "server_1",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "python",
		Enabled:   true,
	}
	if err := registry.Register(config); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	servers, err := registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(servers))
	}

	server, err := registry.Get("server_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if server.Name != "Test Server" {
		t.Fatalf("Expected server name Test Server, got %s", server.Name)
	}

	if err := registry.Unregister("server_1"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	servers, err = registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("Expected 0 servers after unregister, got %d", len(servers))
	}

	if _, err := registry.Get("server_1"); err == nil {
		t.Fatal("Expected error getting unregistered server")
	}
}

func TestInMemoryServerRegistryDuplicate(t *testing.T) {
	registry := NewInMemoryServerRegistry()

	config1 := &MCPServerConfig{
		ID:   "server_1",
		Name: "Server One",
	}
	if err := registry.Register(config1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	config2 := &MCPServerConfig{
		ID:   "server_1",
		Name: "Server One Updated",
	}
	if err := registry.Register(config2); err != nil {
		t.Fatalf("Register duplicate failed: %v", err)
	}

	server, err := registry.Get("server_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if server.Name != "Server One Updated" {
		t.Fatalf("Duplicate register should overwrite, got %s", server.Name)
	}

	servers, err := registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("Expected 1 server after duplicate register, got %d", len(servers))
	}
}

func TestInMemoryToolDiscovery(t *testing.T) {
	registry := NewInMemoryServerRegistry()
	discovery := NewInMemoryToolDiscovery(registry)

	config := &MCPServerConfig{
		ID:   "server_1",
		Name: "Test Server",
		URL:  "",
	}
	registry.Register(config)

	tools, err := discovery.DiscoverTools(context.Background(), "server_1")
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}
	if len(tools) != 0 {
		t.Fatalf("Expected 0 tools for server without URL, got %d", len(tools))
	}

	_, err = discovery.DiscoverTools(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent server")
	}
}

func TestInMemoryToolInvoker(t *testing.T) {
	registry := NewInMemoryServerRegistry()
	invoker := NewInMemoryToolInvoker(registry)

	config := &MCPServerConfig{
		ID:   "server_1",
		Name: "Test Server",
		URL:  "",
	}
	registry.Register(config)

	result, err := invoker.CallTool(context.Background(), "server_1", "test_tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if string(result) != "{}" {
		t.Fatalf("Expected empty result for empty URL, got %s", string(result))
	}
}
