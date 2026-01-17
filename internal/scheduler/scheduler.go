package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a scheduled task
type Task struct {
	ID          string
	Name        string
	Schedule    string // Cron expression
	Command     string
	Args        []string
	Enabled     bool
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int
	FailCount   int
	LastError   string
}

// TaskFunc is a function that can be scheduled
type TaskFunc func(ctx context.Context) error

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks    map[string]*scheduledTask
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type scheduledTask struct {
	task     *Task
	fn       TaskFunc
	ticker   *time.Ticker
	stopChan chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		tasks:  make(map[string]*scheduledTask),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(task *Task, fn TaskFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	// Parse schedule
	interval, err := parseSchedule(task.Schedule)
	if err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	st := &scheduledTask{
		task:     task,
		fn:       fn,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
	}

	s.tasks[task.ID] = st

	// Calculate next run
	task.NextRun = time.Now().Add(interval)

	// Start task if enabled
	if task.Enabled {
		s.wg.Add(1)
		go s.runTask(st)
	}

	return nil
}

// RemoveTask removes a task from the scheduler
func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Stop the task
	st.ticker.Stop()
	close(st.stopChan)
	delete(s.tasks, taskID)

	return nil
}

// EnableTask enables a task
func (s *Scheduler) EnableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if !st.task.Enabled {
		st.task.Enabled = true
		s.wg.Add(1)
		go s.runTask(st)
	}

	return nil
}

// DisableTask disables a task
func (s *Scheduler) DisableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if st.task.Enabled {
		st.task.Enabled = false
		close(st.stopChan)
		st.stopChan = make(chan struct{})
	}

	return nil
}

// GetTask returns a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return st.task, nil
}

// ListTasks returns all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, st := range s.tasks {
		tasks = append(tasks, st.task)
	}

	return tasks
}

// runTask runs a scheduled task
func (s *Scheduler) runTask(st *scheduledTask) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-st.stopChan:
			return
		case <-st.ticker.C:
			if !st.task.Enabled {
				continue
			}

			// Update last run time
			st.task.LastRun = time.Now()
			st.task.RunCount++

			// Execute task
			if err := st.fn(s.ctx); err != nil {
				st.task.FailCount++
				st.task.LastError = err.Error()
			} else {
				st.task.LastError = ""
			}

			// Calculate next run
			interval, _ := parseSchedule(st.task.Schedule)
			st.task.NextRun = time.Now().Add(interval)
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, st := range s.tasks {
		st.ticker.Stop()
		close(st.stopChan)
	}
}

// parseSchedule parses a schedule string into a duration
// Supports simple formats: "1h", "30m", "1d", "@hourly", "@daily", "@weekly"
func parseSchedule(schedule string) (time.Duration, error) {
	switch schedule {
	case "@hourly":
		return time.Hour, nil
	case "@daily":
		return 24 * time.Hour, nil
	case "@weekly":
		return 7 * 24 * time.Hour, nil
	case "@monthly":
		return 30 * 24 * time.Hour, nil
	}

	// Try parsing as duration
	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule format: %s", schedule)
	}

	if duration < time.Minute {
		return 0, fmt.Errorf("schedule must be at least 1 minute")
	}

	return duration, nil
}

// Config represents scheduler configuration
type Config struct {
	Tasks []*TaskConfig `yaml:"tasks"`
}

// TaskConfig represents a task configuration
type TaskConfig struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Schedule string   `yaml:"schedule"`
	Command  string   `yaml:"command"`
	Args     []string `yaml:"args"`
	Enabled  bool     `yaml:"enabled"`
}

// LoadConfig loads scheduler configuration
func LoadConfig(tasks []*TaskConfig) (*Config, error) {
	config := &Config{
		Tasks: make([]*TaskConfig, 0),
	}

	for _, task := range tasks {
		// Validate schedule
		if _, err := parseSchedule(task.Schedule); err != nil {
			return nil, fmt.Errorf("invalid schedule for task %s: %w", task.ID, err)
		}

		config.Tasks = append(config.Tasks, task)
	}

	return config, nil
}
