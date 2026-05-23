package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultComposerState(t *testing.T) {
	state := defaultComposerState()
	assert.NotNil(t, state)
	assert.Len(t, state.Agents, 1)
	assert.Equal(t, "coordinator", state.Agents[0].Name)
	assert.True(t, state.Agents[0].Selected)
	assert.Equal(t, "openai/gpt-5.5 (xhigh)", state.Agents[0].Model)
}

func TestDefaultModelForAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		want      string
	}{
		{name: "coordinatorUsesXHigh", agentName: "coordinator", want: "openai/gpt-5.5 (xhigh)"},
		{name: "workerUsesLow", agentName: "backend-go-developer", want: "openai/gpt-5.5 (low)"},
		{name: "reviewerUsesLow", agentName: "go-readability-reviewer", want: "openai/gpt-5.5 (low)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultModelForAgent(tt.agentName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractDescriptionFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "simpleDescription",
			body:     "# My Project\n\nThis is a description.\n\n## Tasks\n\n- Task 1\n- Task 2",
			expected: "My Project\n\nThis is a description.",
		},
		{
			name:     "descriptionWithMultipleSections",
			body:     "# My Project\n\nThis is a description.\n\nMore details here.\n\n## Tasks\n\n- Task 1",
			expected: "My Project\n\nThis is a description.\n\nMore details here.",
		},
		{
			name:     "descriptionWithSafetyAnalysisSection",
			body:     "# My Project\n\nDescription.\n\n## Safety Analysis\n\n- The coordinator must load/use `stpa-overview`.\n\n## Tasks\n\n- Task 1",
			expected: "My Project\n\nDescription.",
		},
		{
			name:     "noTasksSection",
			body:     "# My Project\n\nThis is a description.",
			expected: "My Project\n\nThis is a description.",
		},
		{
			name:     "emptyBody",
			body:     "",
			expected: "",
		},
		{
			name:     "onlyTasksSection",
			body:     "## Tasks\n\n- Task 1\n- Task 2",
			expected: "",
		},
		{
			name:     "titleWithTrailingWhitespace",
			body:     "# Title Only\n   ",
			expected: "Title Only",
		},
		{
			name:     "tasksSectionAtEnd",
			body:     "# My Project\n\nDescription here.\n\n## Tasks\n\n- Task 1\n\n## Another Section\n\nMore content",
			expected: "My Project\n\nDescription here.\n\n## Another Section\n\nMore content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescriptionFromBody(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTasksFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "simpleTasks",
			body:     "# My Project\n\nDescription.\n\n## Tasks\n\n- Task 1\n- Task 2",
			expected: "- Task 1\n- Task 2",
		},
		{
			name:     "tasksWithSubsections",
			body:     "# My Project\n\nDescription.\n\n## Tasks\n\n- Task 1\n  - Subtask 1\n- Task 2",
			expected: "- Task 1\n  - Subtask 1\n- Task 2",
		},
		{
			name:     "noTasksSection",
			body:     "# My Project\n\nDescription.",
			expected: "",
		},
		{
			name:     "emptyBody",
			body:     "",
			expected: "",
		},
		{
			name:     "tasksFollowedByOtherSection",
			body:     "# My Project\n\nDescription.\n\n## Tasks\n\n- Task 1\n\n## Notes\n\nSome notes",
			expected: "- Task 1",
		},
		{
			name:     "multipleTaskSections",
			body:     "# My Project\n\nDescription.\n\n## Tasks\n\n- Task 1\n\n## Notes\n\nNotes here.\n\n## Task\n\n- Task 2",
			expected: "- Task 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTasksFromBody(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadComposerStateFromDisk(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, composerState)
	}{
		{
			name: "loadFromValidGOAL",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "agent1" -> "agent2"
models:
  "coordinator": "model1"
  "agent1": "model2"
completionGateScript: make test
---
# My Project

Description here.

## Tasks

- Task 1
- Task 2
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.Equal(t, "My Project\n\nDescription here.", state.Description)
				assert.Equal(t, "make test", state.CompletionGate)
				assert.False(t, state.Retrospective)
				assert.Contains(t, state.Flow, "agent1")
				assert.Contains(t, state.Tasks, "Task 1")
				assert.Len(t, state.Agents, 2)
				assert.Equal(t, "agent1", state.Agents[0].Name)
				assert.True(t, state.Agents[0].Selected)
				assert.Equal(t, "model2", state.Agents[0].Model)
			},
		},
		{
			name: "loadSafetyAnalysisFromGOALBody",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "model1"
---
# My Project

Description here.

## Safety Analysis

- The coordinator must load/use ` + "`stpa-overview`" + ` when safety concerns are relevant.

## Tasks

- Task 1
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.True(t, state.SafetyAnalysis)
				assert.Equal(t, "My Project\n\nDescription here.", state.Description)
				assert.Equal(t, "- Task 1", state.Tasks)
			},
		},
		{
			name: "migrateLegacySTPAAgentToSafetyAnalysis",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "backend-go-developer" -> "stpa-analyst"
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "model1"
  "backend-go-developer": "model2"
  "stpa-analyst": "model3"
---
# My Project

Description here.
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.True(t, state.SafetyAnalysis)
				assert.NotContains(t, state.Flow, "stpa-analyst")
				for _, agent := range state.Agents {
					assert.NotEqual(t, "stpa-analyst", agent.Name)
				}
				content := buildGOALContent(state)
				assert.NotContains(t, content, "stpa-analyst\":")
				assert.Contains(t, content, "stpa-overview")
			},
		},
		{
			name: "loadFromMissingGOAL",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, state composerState) {
				assert.Len(t, state.Agents, 1)
				assert.Equal(t, "coordinator", state.Agents[0].Name)
			},
		},
		{
			name: "loadFromGOALWithInvalidFrontmatter",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
invalid yaml content
---
# My Project
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.Len(t, state.Agents, 1)
				assert.Equal(t, "coordinator", state.Agents[0].Name)
			},
		},
		{
			name: "loadFromGOALWithNoModels",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "agent1" -> "agent2"
---
# My Project
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.Len(t, state.Agents, 0)
			},
		},
		{
			name: "loadFromGOALWithRetrospectiveEnabled",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
retrospective: true
---
# My Project
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, state composerState) {
				assert.True(t, state.Retrospective)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)

			state := loadComposerStateFromDisk(dir)

			if tt.validate != nil {
				tt.validate(t, state)
			}
		})
	}
}

func TestBuildGOALContent(t *testing.T) {
	tests := []struct {
		name     string
		state    composerState
		validate func(*testing.T, string)
	}{
		{
			name: "buildCompleteGOAL",
			state: composerState{
				Description:    "My Project\n\nDescription here.",
				CompletionGate: "make test",
				Flow:           `"agent1" -> "agent2"`,
				Tasks:          "- Task 1\n- Task 2",
				Agents: []composerAgentConf{
					{Name: "agent1", Selected: true, Model: "model1"},
					{Name: "agent2", Selected: true, Model: "model2"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "---")
				assert.Contains(t, content, "flow: |")
				assert.Contains(t, content, `"agent1" -> "agent2"`)
				assert.Contains(t, content, "models:")
				assert.Contains(t, content, `"agent1": "model1"`)
				assert.Contains(t, content, `"agent2": "model2"`)
				assert.Contains(t, content, "completionGateScript: make test")
				assert.Contains(t, content, "My Project")
				assert.Contains(t, content, "## Tasks")
				assert.Contains(t, content, "Task 1")
			},
		},
		{
			name: "buildGOALWithSafetyAnalysis",
			state: composerState{
				Description:    "My Project",
				SafetyAnalysis: true,
				Flow:           `"backend-go-developer" -> "go-readability-reviewer"`,
				Agents: []composerAgentConf{
					{Name: "coordinator", Selected: true, Model: "model1"},
					{Name: "backend-go-developer", Selected: true, Model: "model2"},
					{Name: "go-readability-reviewer", Selected: true, Model: "model3"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, `"stpa-analyst" ->`)
				assert.NotContains(t, content, `-> "stpa-analyst"`)
				assert.NotContains(t, content, `"stpa-analyst":`)
				assert.Contains(t, content, "## Safety Analysis")
				assert.Contains(t, content, "stpa-overview")
				assert.Contains(t, content, "coordinator must load/use")
				assert.Contains(t, content, "*-reviewer")
			},
		},
		{
			name: "buildGOALFiltersRetiredSTPAAgent",
			state: composerState{
				Description: "My Project",
				Flow:        `"backend-go-developer" -> "stpa-analyst"`,
				Agents: []composerAgentConf{
					{Name: "stpa-analyst", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, "stpa-analyst")
				assert.NotContains(t, content, "models:")
				assert.NotContains(t, content, "flow:")
			},
		},
		{
			name:  "buildGOALOmitsRetrospectiveByDefault",
			state: composerState{Description: "My Project"},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, "retrospective:")
			},
		},
		{
			name:  "buildGOALIncludesRetrospectiveWhenEnabled",
			state: composerState{Description: "My Project", Retrospective: true},
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "retrospective: true")
			},
		},
		{
			name: "buildGOALWithNoFlow",
			state: composerState{
				Description: "My Project",
				Agents: []composerAgentConf{
					{Name: "agent1", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, "flow:")
				assert.Contains(t, content, "models:")
				assert.Contains(t, content, "My Project")
			},
		},
		{
			name: "buildGOALWithNoAgents",
			state: composerState{
				Description: "My Project",
				Flow:        `"agent1" -> "agent2"`,
			},
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "flow:")
				assert.NotContains(t, content, "models:")
				assert.Contains(t, content, "My Project")
			},
		},
		{
			name: "buildGOALWithUnselectedAgents",
			state: composerState{
				Description: "My Project",
				Agents: []composerAgentConf{
					{Name: "agent1", Selected: false, Model: "model1"},
					{Name: "agent2", Selected: true, Model: "model2"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "models:")
				assert.NotContains(t, content, `"agent1"`)
				assert.Contains(t, content, `"agent2"`)
			},
		},
		{
			name: "buildGOALWithNoTasks",
			state: composerState{
				Description: "My Project",
				Agents: []composerAgentConf{
					{Name: "agent1", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, "## Tasks")
			},
		},
		{
			name: "buildGOALWithNoCompletionGate",
			state: composerState{
				Description: "My Project",
				Agents: []composerAgentConf{
					{Name: "agent1", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.NotContains(t, content, "completionGateScript")
			},
		},
		{
			name: "buildGOALUsesDefaultsForMissingSelectedAgentModels",
			state: composerState{
				Description: "My Project",
				Agents: []composerAgentConf{
					{Name: "coordinator", Selected: true},
					{Name: "builder", Selected: true},
					{Name: "reviewer", Selected: false},
				},
			},
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "models:")
				assert.Contains(t, content, `"coordinator": "openai/gpt-5.5 (xhigh)"`)
				assert.Contains(t, content, `"builder": "openai/gpt-5.5 (low)"`)
				assert.NotContains(t, content, `"reviewer"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildGOALContent(tt.state)
			if tt.validate != nil {
				tt.validate(t, content)
			}
		})
	}
}

func TestDefaultWizardState(t *testing.T) {
	state := defaultWizardState()
	assert.Equal(t, 1, state.CurrentStep)
	assert.False(t, state.SafetyAnalysis)
	assert.Empty(t, state.FromTemplate)
	assert.Empty(t, state.Description)
	assert.Empty(t, state.TechStack)
	assert.Empty(t, state.CompletionGate)
}

func TestSyncWizardState(t *testing.T) {
	tests := []struct {
		name     string
		wizard   wizardState
		state    composerState
		validate func(*testing.T, wizardState)
	}{
		{
			name: "syncWithEmptyTechStackDoesNotInferSafetyAnalysisFromRetiredAgent",
			wizard: wizardState{
				CurrentStep: 1,
				TechStack:   []string{},
			},
			state: composerState{
				Agents: []composerAgentConf{
					{Name: "go", Selected: true, Model: "model1"},
					{Name: "stpa-analyst", Selected: true, Model: "model2"},
				},
			},
			validate: func(t *testing.T, wizard wizardState) {
				assert.Contains(t, wizard.TechStack, "go")
				assert.False(t, wizard.SafetyAnalysis)
			},
		},
		{
			name: "syncWithExistingTechStack",
			wizard: wizardState{
				CurrentStep: 1,
				TechStack:   []string{"react"},
			},
			state: composerState{
				Agents: []composerAgentConf{
					{Name: "go", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, wizard wizardState) {
				assert.Equal(t, []string{"react"}, wizard.TechStack)
			},
		},
		{
			name: "syncWithExistingSafetyAnalysis",
			wizard: wizardState{
				CurrentStep:    1,
				SafetyAnalysis: true,
			},
			state: composerState{
				Agents: []composerAgentConf{
					{Name: "go", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, wizard wizardState) {
				assert.True(t, wizard.SafetyAnalysis)
			},
		},
		{
			name: "syncWithNoStpaAnalyst",
			wizard: wizardState{
				CurrentStep: 1,
			},
			state: composerState{
				Agents: []composerAgentConf{
					{Name: "go", Selected: true, Model: "model1"},
				},
			},
			validate: func(t *testing.T, wizard wizardState) {
				assert.False(t, wizard.SafetyAnalysis)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncWizardState(tt.wizard, tt.state)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestWorkflowTemplatesDoNotRouteSafetyAnalysisToSTPAAgent(t *testing.T) {
	for _, tmpl := range workflowTemplates {
		t.Run(tmpl.ID, func(t *testing.T) {
			assert.NotContains(t, tmpl.Flow, "stpa-analyst")
			assert.NotContains(t, tmpl.Flow, "coordinator")
			for _, agent := range tmpl.Agents {
				assert.NotEqual(t, "stpa-analyst", agent.Name)
			}
		})
	}
}

func TestWorkflowTemplateAgentsExistInSkeleton(t *testing.T) {
	for _, tmpl := range workflowTemplates {
		t.Run(tmpl.ID, func(t *testing.T) {
			for _, agent := range tmpl.Agents {
				agentPath := filepath.Join("skel", ".sgai", "agent", agent.Name+".md")
				_, errStat := os.Stat(agentPath)
				require.NoError(t, errStat, "template agent %s must exist at %s", agent.Name, agentPath)
			}
		})
	}
}

func TestWorkflowTemplatesUseDefaultModels(t *testing.T) {
	for _, tmpl := range workflowTemplates {
		t.Run(tmpl.ID, func(t *testing.T) {
			for _, agent := range tmpl.Agents {
				want := "openai/gpt-5.5 (low)"
				if agent.Name == "coordinator" {
					want = "openai/gpt-5.5 (xhigh)"
				}
				assert.Equal(t, want, agent.Model, "agent %s", agent.Name)
			}
		})
	}
}

func TestTechStackFromAgents(t *testing.T) {
	tests := []struct {
		name     string
		agents   []composerAgentConf
		expected []string
	}{
		{
			name: "extractGoStack",
			agents: []composerAgentConf{
				{Name: "go", Selected: true, Model: "model1"},
			},
			expected: []string{"go"},
		},
		{
			name: "extractReactStack",
			agents: []composerAgentConf{
				{Name: "react", Selected: true, Model: "model1"},
			},
			expected: []string{"react"},
		},
		{
			name: "extractHTMXStack",
			agents: []composerAgentConf{
				{Name: "htmx-picocss", Selected: true, Model: "model1"},
			},
			expected: []string{"htmx"},
		},
		{
			name: "extractShellStack",
			agents: []composerAgentConf{
				{Name: "shell-script", Selected: true, Model: "model1"},
			},
			expected: []string{"shell"},
		},
		{
			name: "extractGeneralPurposeStack",
			agents: []composerAgentConf{
				{Name: "general-purpose", Selected: true, Model: "model1"},
			},
			expected: []string{"general-purpose"},
		},
		{
			name: "extractClaudeSDKStack",
			agents: []composerAgentConf{
				{Name: "agent-sdk-verifier-ts", Selected: true, Model: "model1"},
			},
			expected: []string{"claudesdk"},
		},
		{
			name: "extractOpenAISDKStack",
			agents: []composerAgentConf{
				{Name: "openai-sdk-verifier-py", Selected: true, Model: "model1"},
			},
			expected: []string{"openaisdk"},
		},
		{
			name: "extractMultipleStacks",
			agents: []composerAgentConf{
				{Name: "go", Selected: true, Model: "model1"},
				{Name: "react", Selected: true, Model: "model2"},
			},
			expected: []string{"go", "react"},
		},
		{
			name: "extractWithUnselectedAgents",
			agents: []composerAgentConf{
				{Name: "go", Selected: true, Model: "model1"},
				{Name: "react", Selected: false, Model: "model2"},
			},
			expected: []string{"go"},
		},
		{
			name:     "extractFromEmptyAgents",
			agents:   []composerAgentConf{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := techStackFromAgents(tt.agents)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgentSelected(t *testing.T) {
	tests := []struct {
		name      string
		agents    []composerAgentConf
		agentName string
		expected  bool
	}{
		{
			name: "agentIsSelected",
			agents: []composerAgentConf{
				{Name: "agent1", Selected: true, Model: "model1"},
			},
			agentName: "agent1",
			expected:  true,
		},
		{
			name: "agentIsNotSelected",
			agents: []composerAgentConf{
				{Name: "agent1", Selected: false, Model: "model1"},
			},
			agentName: "agent1",
			expected:  false,
		},
		{
			name: "agentNotFound",
			agents: []composerAgentConf{
				{Name: "agent1", Selected: true, Model: "model1"},
			},
			agentName: "agent2",
			expected:  false,
		},
		{
			name:      "emptyAgents",
			agents:    []composerAgentConf{},
			agentName: "agent1",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentSelected(tt.agents, tt.agentName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetComposerSession(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*testing.T, string)
		validate   func(*testing.T, *composerSession)
		checkReuse bool
	}{
		{
			name: "createNewSession",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, session *composerSession) {
				t.Helper()
				assert.NotNil(t, session)
				assert.NotNil(t, session.state)
				assert.NotNil(t, session.wizard)
				assert.Len(t, session.state.Agents, 1)
				assert.Equal(t, "coordinator", session.state.Agents[0].Name)
			},
		},
		{
			name: "loadExistingSession",
			setupFunc: func(t *testing.T, dir string) {
				t.Helper()
				goalContent := `---
flow: |
  "agent1" -> "agent2"
models:
  "coordinator": "model1"
  "agent1": "model2"
---
# My Project
`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, session *composerSession) {
				t.Helper()
				assert.NotNil(t, session)
				assert.NotNil(t, session.state)
				assert.Contains(t, session.state.Flow, "agent1")
				assert.Len(t, session.state.Agents, 2)
			},
		},
		{
			name: "reuseExistingSession",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, session *composerSession) {
				t.Helper()
				assert.NotNil(t, session)
			},
			checkReuse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)
			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			session1 := server.getComposerSession(workspacePath)
			if tt.validate != nil {
				tt.validate(t, session1)
			}

			if tt.checkReuse {
				session2 := server.getComposerSession(workspacePath)
				assert.Equal(t, session1, session2, "Should return the same session instance")
			}
		})
	}
}
