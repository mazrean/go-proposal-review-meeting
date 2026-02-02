package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// workflowProjectRoot returns the main project root (first module when using go.work)
func workflowProjectRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}
	// Handle go.work case: take only the first line (main project)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines[0]
}

// WorkflowTrigger represents the 'on' section of a GitHub Actions workflow
type WorkflowTrigger struct {
	WorkflowDispatch map[string]interface{} `yaml:"workflow_dispatch"`
	Schedule         []ScheduleItem         `yaml:"schedule"`
}

// ScheduleItem represents a single cron schedule
type ScheduleItem struct {
	Cron string `yaml:"cron"`
}

// Workflow represents a GitHub Actions workflow file structure
type Workflow struct {
	On   WorkflowTrigger `yaml:"on"`
	Jobs map[string]Job  `yaml:"jobs"`
	Name string          `yaml:"name"`
}

// Job represents a job in the workflow
type Job struct {
	RunsOn string `yaml:"runs-on"`
	Steps  []Step `yaml:"steps"`
}

// Step represents a step in a job
type Step struct {
	Name string `yaml:"name"`
	Uses string `yaml:"uses"`
	Run  string `yaml:"run"`
}

func TestWeeklyUpdateWorkflowExists(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Error(".github/workflows/weekly-update.yml should exist")
		return
	}
	if err != nil {
		t.Errorf("error checking workflow file: %v", err)
		return
	}
	if info.IsDir() {
		t.Error("weekly-update.yml should be a file, not a directory")
	}
}

func TestWeeklyUpdateWorkflowCronSchedule(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	// Check cron schedule
	// Thursday 12:00 JST = Thursday 03:00 UTC
	// Cron format: minute hour day month weekday
	// 0 3 * * 4 = at 03:00 on Thursday
	if len(workflow.On.Schedule) == 0 {
		t.Error("workflow should have a schedule trigger")
		return
	}

	expectedCron := "0 3 * * 4"
	found := false
	for _, sched := range workflow.On.Schedule {
		if strings.TrimSpace(sched.Cron) == expectedCron {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("workflow should have cron schedule '%s' (Thursday 12:00 JST), got: %v",
			expectedCron, workflow.On.Schedule)
	}
}

func TestWeeklyUpdateWorkflowDispatch(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	// Check that workflow_dispatch is present (even if empty)
	if workflow.On.WorkflowDispatch == nil {
		// Check raw YAML for workflow_dispatch key
		if !strings.Contains(string(content), "workflow_dispatch") {
			t.Error("workflow should have workflow_dispatch trigger for manual execution")
		}
	}
}

func TestWeeklyUpdateWorkflowHasUpdateJob(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	// Check that there's at least one job
	if len(workflow.Jobs) == 0 {
		t.Error("workflow should have at least one job defined")
	}

	// Check for update job
	if _, exists := workflow.Jobs["update"]; !exists {
		t.Error("workflow should have an 'update' job")
	}
}

func TestWeeklyUpdateWorkflowName(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	if workflow.Name == "" {
		t.Error("workflow should have a name")
	}
}

// Task 6.6: Tests for build step implementation
func TestWeeklyUpdateWorkflowHasBuildStep(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	updateJob, exists := workflow.Jobs["update"]
	if !exists {
		t.Fatal("workflow should have an 'update' job")
	}

	// Check for Build static site step
	hasBuildStep := false
	for _, step := range updateJob.Steps {
		if step.Name == "Build static site" {
			hasBuildStep = true
			break
		}
	}

	if !hasBuildStep {
		t.Error("workflow should have 'Build static site' step")
	}
}

func TestWeeklyUpdateWorkflowBuildStepRunsMakeBuild(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	updateJob := workflow.Jobs["update"]

	// Find build step and verify it runs make build
	for _, step := range updateJob.Steps {
		if step.Name == "Build static site" {
			if !strings.Contains(step.Run, "make build") {
				t.Error("Build static site step should run 'make build'")
			}
			return
		}
	}

	t.Error("Build static site step not found")
}

func TestWeeklyUpdateWorkflowHasNodeSetup(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	updateJob := workflow.Jobs["update"]

	// Check for Node.js setup step
	hasNodeSetup := false
	for _, step := range updateJob.Steps {
		if step.Uses == "actions/setup-node@v4" {
			hasNodeSetup = true
			break
		}
	}

	if !hasNodeSetup {
		t.Error("workflow should have Node.js setup step using actions/setup-node@v4")
	}
}

func TestWeeklyUpdateWorkflowHasNpmInstall(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}

	updateJob := workflow.Jobs["update"]

	// Check for npm install step
	hasNpmInstall := false
	for _, step := range updateJob.Steps {
		if step.Name == "Install npm dependencies" {
			if strings.Contains(step.Run, "npm ci") {
				hasNpmInstall = true
				break
			}
		}
	}

	if !hasNpmInstall {
		t.Error("workflow should have 'Install npm dependencies' step that runs 'npm ci'")
	}
}

func TestWeeklyUpdateWorkflowBuildStepHasCondition(t *testing.T) {
	root := workflowProjectRoot(t)
	path := filepath.Join(root, ".github/workflows/weekly-update.yml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	// Check that build step has the correct conditional
	// We need to verify: if: steps.parse.outputs.has_changes == 'true'
	if !strings.Contains(string(content), "steps.parse.outputs.has_changes == 'true'") {
		t.Error("Build steps should have conditional 'steps.parse.outputs.has_changes == 'true''")
	}

	// Verify build step section contains the conditional
	buildStepIndex := strings.Index(string(content), "Build static site")
	if buildStepIndex == -1 {
		t.Error("Build static site step not found")
		return
	}

	// Check that there's an 'if' condition before the build step
	nodeSetupIndex := strings.Index(string(content), "Setup Node.js")
	if nodeSetupIndex == -1 {
		t.Error("Setup Node.js step not found")
		return
	}

	// The content between these steps should contain the condition
	if !strings.Contains(string(content)[nodeSetupIndex:], "if: steps.parse.outputs.has_changes == 'true'") {
		t.Error("Node.js setup should have change detection condition")
	}
}
