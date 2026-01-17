package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xuanyiying/cleanup-cli/internal/scheduler"
)

var (
	scheduleID       string
	scheduleName     string
	scheduleInterval string
	scheduleCommand  string
	scheduleEnabled  bool
)

// scheduleCmd represents the schedule command
var scheduleCmd = &cobra.Command{
	Use:     "schedule",
	Aliases: []string{"sched", "cron"},
	Short:   "Manage scheduled cleanup tasks",
	Long: `Schedule automatic cleanup tasks to run periodically.

Supported intervals:
  @hourly  - Run every hour
  @daily   - Run every day
  @weekly  - Run every week
  @monthly - Run every month
  1h       - Run every hour
  30m      - Run every 30 minutes
  2h30m    - Run every 2 hours and 30 minutes

Examples:
  cleanup schedule add --id daily-cleanup --name "Daily Cleanup" --interval @daily --command "cleanup organize ~/Downloads"
  cleanup schedule list
  cleanup schedule enable daily-cleanup
  cleanup schedule disable daily-cleanup
  cleanup schedule remove daily-cleanup`,
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new scheduled task",
	RunE:  runScheduleAdd,
}

var scheduleListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all scheduled tasks",
	RunE:    runScheduleList,
}

var scheduleEnableCmd = &cobra.Command{
	Use:   "enable [task-id]",
	Short: "Enable a scheduled task",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleEnable,
}

var scheduleDisableCmd = &cobra.Command{
	Use:   "disable [task-id]",
	Short: "Disable a scheduled task",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleDisable,
}

var scheduleRemoveCmd = &cobra.Command{
	Use:     "remove [task-id]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a scheduled task",
	Args:    cobra.ExactArgs(1),
	RunE:    runScheduleRemove,
}

func init() {
	// Add subcommands
	scheduleCmd.AddCommand(scheduleAddCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleEnableCmd)
	scheduleCmd.AddCommand(scheduleDisableCmd)
	scheduleCmd.AddCommand(scheduleRemoveCmd)

	// Add flags
	scheduleAddCmd.Flags().StringVar(&scheduleID, "id", "", "Task ID (required)")
	scheduleAddCmd.Flags().StringVar(&scheduleName, "name", "", "Task name (required)")
	scheduleAddCmd.Flags().StringVar(&scheduleInterval, "interval", "", "Schedule interval (required)")
	scheduleAddCmd.Flags().StringVar(&scheduleCommand, "command", "", "Command to run (required)")
	scheduleAddCmd.Flags().BoolVar(&scheduleEnabled, "enabled", true, "Enable task immediately")

	scheduleAddCmd.MarkFlagRequired("id")
	scheduleAddCmd.MarkFlagRequired("name")
	scheduleAddCmd.MarkFlagRequired("interval")
	scheduleAddCmd.MarkFlagRequired("command")

	rootCmd.AddCommand(scheduleCmd)
}

func runScheduleAdd(cmd *cobra.Command, args []string) error {
	// Create scheduler
	sched := scheduler.NewScheduler()
	defer sched.Stop()

	// Create task
	task := &scheduler.Task{
		ID:       scheduleID,
		Name:     scheduleName,
		Schedule: scheduleInterval,
		Command:  scheduleCommand,
		Enabled:  scheduleEnabled,
	}

	// Create task function
	taskFn := func(ctx context.Context) error {
		fmt.Printf("Running scheduled task: %s\n", task.Name)
		
		// Execute command
		cmd := exec.CommandContext(ctx, "sh", "-c", task.Command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		return cmd.Run()
	}

	// Add task
	if err := sched.AddTask(task, taskFn); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	fmt.Printf("✓ Task '%s' added successfully\n", task.Name)
	fmt.Printf("  ID: %s\n", task.ID)
	fmt.Printf("  Schedule: %s\n", task.Schedule)
	fmt.Printf("  Command: %s\n", task.Command)
	fmt.Printf("  Enabled: %v\n", task.Enabled)
	fmt.Printf("  Next run: %s\n", task.NextRun.Format("2006-01-02 15:04:05"))

	return nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	// Create scheduler
	sched := scheduler.NewScheduler()
	defer sched.Stop()

	// Get tasks
	tasks := sched.ListTasks()

	if len(tasks) == 0 {
		fmt.Println("No scheduled tasks found")
		return nil
	}

	// Display tasks in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSCHEDULE\tENABLED\tRUN COUNT\tNEXT RUN")
	fmt.Fprintln(w, "---\t----\t--------\t-------\t---------\t--------")

	for _, task := range tasks {
		enabled := "No"
		if task.Enabled {
			enabled = "Yes"
		}

		nextRun := "N/A"
		if !task.NextRun.IsZero() {
			nextRun = task.NextRun.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			task.ID, task.Name, task.Schedule, enabled, task.RunCount, nextRun)
	}

	w.Flush()
	return nil
}

func runScheduleEnable(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Create scheduler
	sched := scheduler.NewScheduler()
	defer sched.Stop()

	if err := sched.EnableTask(taskID); err != nil {
		return fmt.Errorf("failed to enable task: %w", err)
	}

	fmt.Printf("✓ Task '%s' enabled\n", taskID)
	return nil
}

func runScheduleDisable(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Create scheduler
	sched := scheduler.NewScheduler()
	defer sched.Stop()

	if err := sched.DisableTask(taskID); err != nil {
		return fmt.Errorf("failed to disable task: %w", err)
	}

	fmt.Printf("✓ Task '%s' disabled\n", taskID)
	return nil
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Confirm removal
	fmt.Printf("Remove task '%s'? (y/n): ", taskID)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Create scheduler
	sched := scheduler.NewScheduler()
	defer sched.Stop()

	if err := sched.RemoveTask(taskID); err != nil {
		return fmt.Errorf("failed to remove task: %w", err)
	}

	fmt.Printf("✓ Task '%s' removed\n", taskID)
	return nil
}
