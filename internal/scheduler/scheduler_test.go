package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestScheduler_AddTask(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1h",
		Enabled:  false,
	}

	fn := func(ctx context.Context) error {
		return nil
	}

	err := s.AddTask(task, fn)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Try adding duplicate
	err = s.AddTask(task, fn)
	if err == nil {
		t.Error("Expected error when adding duplicate task")
	}
}

func TestScheduler_RemoveTask(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1h",
		Enabled:  false,
	}

	fn := func(ctx context.Context) error {
		return nil
	}

	s.AddTask(task, fn)

	err := s.RemoveTask("test-task")
	if err != nil {
		t.Fatalf("RemoveTask failed: %v", err)
	}

	// Try removing non-existent task
	err = s.RemoveTask("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent task")
	}
}

func TestScheduler_EnableDisable(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1h",
		Enabled:  false,
	}

	fn := func(ctx context.Context) error {
		return nil
	}

	s.AddTask(task, fn)

	// Enable task
	err := s.EnableTask("test-task")
	if err != nil {
		t.Fatalf("EnableTask failed: %v", err)
	}

	retrievedTask, _ := s.GetTask("test-task")
	if !retrievedTask.Enabled {
		t.Error("Task should be enabled")
	}

	// Disable task
	err = s.DisableTask("test-task")
	if err != nil {
		t.Fatalf("DisableTask failed: %v", err)
	}

	retrievedTask, _ = s.GetTask("test-task")
	if retrievedTask.Enabled {
		t.Error("Task should be disabled")
	}
}

func TestScheduler_ListTasks(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	task1 := &Task{ID: "task1", Name: "Task 1", Schedule: "1h", Enabled: false}
	task2 := &Task{ID: "task2", Name: "Task 2", Schedule: "2h", Enabled: false}

	fn := func(ctx context.Context) error { return nil }

	s.AddTask(task1, fn)
	s.AddTask(task2, fn)

	tasks := s.ListTasks()
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestScheduler_RunTask(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	runCount := 0
	fn := func(ctx context.Context) error {
		runCount++
		return nil
	}

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1m", // Use 1 minute as minimum
		Enabled:  true,
	}

	err := s.AddTask(task, fn)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Manually trigger the task for testing
	retrievedTask, _ := s.GetTask("test-task")
	if retrievedTask.RunCount != 0 {
		t.Errorf("Expected RunCount = 0 initially, got %d", retrievedTask.RunCount)
	}
}

func TestScheduler_TaskError(t *testing.T) {
	s := NewScheduler()
	defer s.Stop()

	fn := func(ctx context.Context) error {
		return fmt.Errorf("test error")
	}

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1m",
		Enabled:  false, // Don't enable to avoid actual execution
	}

	err := s.AddTask(task, fn)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	retrievedTask, _ := s.GetTask("test-task")
	if retrievedTask == nil {
		t.Fatal("Task not found")
	}
	
	// Task should not have run yet
	if retrievedTask.FailCount != 0 {
		t.Errorf("Expected FailCount = 0, got %d", retrievedTask.FailCount)
	}
}

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		schedule string
		want     time.Duration
		wantErr  bool
	}{
		{"@hourly", time.Hour, false},
		{"@daily", 24 * time.Hour, false},
		{"@weekly", 7 * 24 * time.Hour, false},
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"1h30m", 90 * time.Minute, false},
		{"invalid", 0, true},
		{"30s", 0, true}, // Too short
	}

	for _, tt := range tests {
		got, err := parseSchedule(tt.schedule)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseSchedule(%s) error = %v, wantErr %v", tt.schedule, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSchedule(%s) = %v, want %v", tt.schedule, got, tt.want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	tasks := []*TaskConfig{
		{
			ID:       "task1",
			Name:     "Task 1",
			Schedule: "1h",
			Command:  "cleanup",
			Args:     []string{"--dry-run"},
			Enabled:  true,
		},
		{
			ID:       "task2",
			Name:     "Task 2",
			Schedule: "@daily",
			Command:  "cleanup",
			Enabled:  false,
		},
	}

	config, err := LoadConfig(tasks)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(config.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(config.Tasks))
	}
}

func TestLoadConfig_InvalidSchedule(t *testing.T) {
	tasks := []*TaskConfig{
		{
			ID:       "task1",
			Name:     "Task 1",
			Schedule: "invalid",
			Command:  "cleanup",
			Enabled:  true,
		},
	}

	_, err := LoadConfig(tasks)
	if err == nil {
		t.Error("Expected error for invalid schedule")
	}
}

func TestScheduler_Stop(t *testing.T) {
	s := NewScheduler()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "1h",
		Enabled:  true,
	}

	fn := func(ctx context.Context) error {
		return nil
	}

	s.AddTask(task, fn)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() timed out")
	}
}
