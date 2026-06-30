package agent

import (
	"testing"
)

func TestNewTaskGraph(t *testing.T) {
	root := &SubTask{Label: "root", Status: "pending"}
	g := NewTaskGraph(root)
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.entry == nil {
		t.Fatal("expected entry node")
	}
	if g.entry.Task.Label != "root" {
		t.Errorf("expected label 'root', got '%s'", g.entry.Task.Label)
	}
}

func TestTaskGraphAddTask(t *testing.T) {
	g := NewTaskGraph(nil)
	err := g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = g.AddTask(SubTask{Label: "task2", Status: "pending"}, []string{"task1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskGraphAddTaskDuplicate(t *testing.T) {
	g := NewTaskGraph(nil)
	_ = g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	err := g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	if err == nil {
		t.Error("expected error for duplicate task")
	}
}

func TestTaskGraphUpdateStatus(t *testing.T) {
	g := NewTaskGraph(nil)
	_ = g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	g.UpdateStatus("task1", "completed")
	// No error expected
}

func TestTaskGraphTopologicalSort(t *testing.T) {
	g := NewTaskGraph(nil)
	_ = g.AddTask(SubTask{Label: "task1", Status: "pending", Index: 0}, []string{})
	_ = g.AddTask(SubTask{Label: "task2", Status: "pending", Index: 1}, []string{"task1"})
	_ = g.AddTask(SubTask{Label: "task3", Status: "pending", Index: 2}, []string{"task1"})
	result := g.TopologicalSort()
	if len(result) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(result))
	}
}

func TestTaskGraphDependencies(t *testing.T) {
	g := NewTaskGraph(nil)
	_ = g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	_ = g.AddTask(SubTask{Label: "task2", Status: "pending"}, []string{"task1"})
	deps, err := g.Dependencies("task2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Label != "task1" {
		t.Errorf("expected dependency 'task1', got '%s'", deps[0].Label)
	}
}

func TestTaskGraphDependenciesNotFound(t *testing.T) {
	g := NewTaskGraph(nil)
	_, err := g.Dependencies("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestTaskGraphSnapshot(t *testing.T) {
	g := NewTaskGraph(nil)
	_ = g.AddTask(SubTask{Label: "task1", Status: "pending"}, []string{})
	snap := g.Snapshot()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snap.Nodes) != 1 {
		t.Errorf("expected 1 node in snapshot, got %d", len(snap.Nodes))
	}
}
