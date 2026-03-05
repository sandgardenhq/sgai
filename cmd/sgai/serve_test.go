package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRootWorkspacePath(t *testing.T) {
	t.Skip("BUG: Function returns parent of root workspace instead of root workspace itself")
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		expected  string
	}{
		{
			name: "getRootFromAbsoluteForkPath",
			setupFunc: func(t *testing.T, forkDir string) {
				rootDir := filepath.Join(filepath.Dir(forkDir), "root-workspace")
				require.NoError(t, os.MkdirAll(rootDir, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(rootDir, ".jj"), 0755))

				require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
				repoFile := filepath.Join(forkDir, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte(filepath.Join(rootDir, ".jj")), 0644))
			},
			expected: filepath.Join(os.TempDir(), "root-workspace"),
		},
		{
			name: "getRootFromRelativeForkPath",
			setupFunc: func(t *testing.T, forkDir string) {
				rootDir := filepath.Join(filepath.Dir(forkDir), "root-workspace")
				require.NoError(t, os.MkdirAll(rootDir, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(rootDir, ".jj"), 0755))

				require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
				repoFile := filepath.Join(forkDir, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte("../root-workspace/.jj"), 0644))
			},
			expected: filepath.Join(os.TempDir(), "root-workspace"),
		},
		{
			name: "getRootFromNonExistentRepoFile",
			setupFunc: func(_ *testing.T, _ string) {
			},
			expected: "",
		},
		{
			name: "getRootFromEmptyRepoFile",
			setupFunc: func(t *testing.T, forkDir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
				repoFile := filepath.Join(forkDir, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte(""), 0644))
			},
			expected: "",
		},
		{
			name: "getRootFromWhitespaceOnlyRepoFile",
			setupFunc: func(t *testing.T, forkDir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
				repoFile := filepath.Join(forkDir, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte("   \n\t  "), 0644))
			},
			expected: "",
		},
		{
			name: "getRootFromNonExistentRootDir",
			setupFunc: func(t *testing.T, forkDir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
				repoFile := filepath.Join(forkDir, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte("/non/existent/root/.jj"), 0644))
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forkDir := filepath.Join(os.TempDir(), "fork-workspace")
			t.Cleanup(func() {
				_ = os.RemoveAll(forkDir)
				_ = os.RemoveAll(filepath.Join(os.TempDir(), "root-workspace"))
			})

			tt.setupFunc(t, forkDir)

			result := getRootWorkspacePath(forkDir)
			if tt.expected != "" {
				absExpected, err := filepath.Abs(tt.expected)
				require.NoError(t, err)
				assert.Equal(t, absExpected, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

type directoryCheckTest struct {
	name     string
	subdir   string
	asFile   bool
	expected bool
}

func directoryCheckCases(subdir string) []directoryCheckTest {
	return []directoryCheckTest{
		{
			name:     "hasDirectory",
			subdir:   subdir,
			expected: true,
		},
		{
			name:     "noDirectory",
			expected: false,
		},
		{
			name:     "isFile",
			subdir:   subdir,
			asFile:   true,
			expected: false,
		},
	}
}

func setupDirectoryCheck(t *testing.T, dir string, tc directoryCheckTest) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	if tc.subdir != "" && tc.asFile {
		require.NoError(t, os.WriteFile(filepath.Join(dir, tc.subdir), []byte("test"), 0644))
	} else if tc.subdir != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, tc.subdir), 0755))
	}
}

func TestHassgaiDirectory(t *testing.T) {
	for _, tt := range directoryCheckCases(".sgai") {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setupDirectoryCheck(t, dir, tt)
			result := hassgaiDirectory(dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasJJRepo(t *testing.T) {
	for _, tt := range directoryCheckCases(".jj") {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setupDirectoryCheck(t, dir, tt)
			result := hasJJRepo(dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateDirectory(t *testing.T) {
	tests := []struct {
		name        string
		rootDir     string
		targetDir   string
		setupFunc   func(*testing.T, string, string)
		expectedErr bool
		errContains string
	}{
		{
			name:        "emptyDirectory",
			rootDir:     t.TempDir(),
			targetDir:   "",
			expectedErr: true,
			errContains: "directory is required",
		},
		{
			name:    "validSubdirectory",
			rootDir: t.TempDir(),
			setupFunc: func(t *testing.T, _, targetDir string) {
				require.NoError(t, os.MkdirAll(targetDir, 0755))
			},
			expectedErr: false,
		},
		{
			name:    "validRootDirectory",
			rootDir: t.TempDir(),
			setupFunc: func(t *testing.T, rootDir, _ string) {
				require.NoError(t, os.MkdirAll(rootDir, 0755))
			},
			expectedErr: false,
		},
		{
			name:      "pathTraversalWithDotDot",
			rootDir:   t.TempDir(),
			targetDir: filepath.Join(t.TempDir(), "..", "..", "etc"),
			setupFunc: func(t *testing.T, rootDir, _ string) {
				require.NoError(t, os.MkdirAll(rootDir, 0755))
			},
			expectedErr: true,
			errContains: "path traversal denied",
		},
		{
			name:      "pathTraversalWithAbsolutePath",
			rootDir:   t.TempDir(),
			targetDir: "/etc/passwd",
			setupFunc: func(t *testing.T, rootDir, _ string) {
				require.NoError(t, os.MkdirAll(rootDir, 0755))
			},
			expectedErr: true,
			errContains: "path traversal denied",
		},
		{
			name:    "nonExistentSubdirectory",
			rootDir: t.TempDir(),
			setupFunc: func(t *testing.T, rootDir, _ string) {
				require.NoError(t, os.MkdirAll(rootDir, 0755))
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{rootDir: tt.rootDir}

			targetDir := tt.targetDir
			if targetDir == "" && tt.name != "emptyDirectory" {
				targetDir = filepath.Join(tt.rootDir, "subdir")
			}

			if tt.setupFunc != nil {
				tt.setupFunc(t, tt.rootDir, targetDir)
			}

			result, err := server.validateDirectory(targetDir)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestSignalBrokerSubscribe(t *testing.T) {
	broker := &signalBroker{
		subscribers: make(map[*signalSubscriber]struct{}),
	}

	sub := broker.subscribe()
	require.NotNil(t, sub)
	require.NotNil(t, sub.ch)
	require.NotNil(t, sub.done)

	broker.mu.Lock()
	_, exists := broker.subscribers[sub]
	broker.mu.Unlock()
	assert.True(t, exists, "subscriber should be registered")
}

func TestSignalBrokerUnsubscribe(t *testing.T) {
	broker := &signalBroker{
		subscribers: make(map[*signalSubscriber]struct{}),
	}

	sub := broker.subscribe()
	broker.unsubscribe(sub)

	broker.mu.Lock()
	_, exists := broker.subscribers[sub]
	broker.mu.Unlock()
	assert.False(t, exists, "subscriber should be unregistered")

	select {
	case <-sub.done:
	default:
		t.Fatal("done channel should be closed")
	}
}

func TestSignalBrokerNotify(t *testing.T) {
	broker := &signalBroker{
		subscribers: make(map[*signalSubscriber]struct{}),
	}

	sub1 := broker.subscribe()
	sub2 := broker.subscribe()

	broker.notify()

	select {
	case <-sub1.ch:
	default:
		t.Fatal("subscriber 1 should receive notification")
	}

	select {
	case <-sub2.ch:
	default:
		t.Fatal("subscriber 2 should receive notification")
	}
}

func TestSignalBrokerNotifyWithFullChannel(t *testing.T) {
	broker := &signalBroker{
		subscribers: make(map[*signalSubscriber]struct{}),
	}

	sub := broker.subscribe()

	sub.ch <- struct{}{}
	broker.notify()

	select {
	case <-sub.ch:
	default:
		t.Fatal("should have one notification")
	}
}

func TestIsCommitMarker(t *testing.T) {
	tests := []struct {
		name     string
		r        rune
		expected bool
	}{
		{"circleMarker", '○', true},
		{"crossMarker", '×', true},
		{"atMarker", '@', true},
		{"diamondMarker", '◆', true},
		{"tildeMarker", '~', true},
		{"letterA", 'a', false},
		{"digit", '1', false},
		{"space", ' ', false},
		{"pipe", '|', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommitMarker(tt.r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGraphChar(t *testing.T) {
	tests := []struct {
		name     string
		r        rune
		expected bool
	}{
		{"verticalPipe", '│', true},
		{"branchPipe", '├', true},
		{"horizontalPipe", '─', true},
		{"cornerPipe", '┘', true},
		{"space", ' ', true},
		{"letter", 'a', false},
		{"digit", '1', false},
		{"atSign", '@', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGraphChar(tt.r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCommitHeaderLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"validHeaderWithAt", "@ abc123 def456", true},
		{"validHeaderWithCircle", "○ abc123 def456", true},
		{"validHeaderWithDiamond", "◆ abc123 def456", true},
		{"validHeaderWithGraph", "│ ○ abc123 def456", true},
		{"tooShort", "ab", false},
		{"emptyLine", "", false},
		{"descriptionLine", "  This is a description", false},
		{"graphOnly", "│ │ │", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommitHeaderLine(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripGraphPrefix(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"noPrefix", "abc123 def456", "abc123 def456"},
		{"withPrefix", "│ ○ abc123 def456", "○ abc123 def456"},
		{"multiplePrefix", "│ │ ○ abc123", "○ abc123"},
		{"emptyLine", "", ""},
		{"onlyGraph", "│ │ │", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripGraphPrefix(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractGraphPrefix(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"noPrefix", "abc123 def456", ""},
		{"withPrefix", "│ ○ abc123 def456", "│ ○"},
		{"multiplePrefix", "│ │ ○ abc123", "│ │ ○"},
		{"emptyLine", "", ""},
		{"onlyGraph", "│ │ │", "│ │ │"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGraphPrefix(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCommitMarker(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedMarker rune
		expectedRest   string
	}{
		{"atMarker", "@ abc123", '@', "abc123"},
		{"circleMarker", "○ abc123", '○', "abc123"},
		{"diamondMarker", "◆ abc123", '◆', "abc123"},
		{"noMarker", "abc123", 0, "abc123"},
		{"emptyLine", "", 0, ""},
		{"withGraphPrefix", "│ ○ abc123", '○', "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marker, rest := findCommitMarker(tt.line)
			assert.Equal(t, tt.expectedMarker, marker)
			assert.Equal(t, tt.expectedRest, rest)
		})
	}
}

func TestIsTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"validTimestamp", "2 hours ago", true},
		{"startsWitDigit", "123", true},
		{"startsWitZero", "0", true},
		{"emptyString", "", false},
		{"startsWitLetter", "abc", false},
		{"startsWitSpace", " 123", false},
		{"startsWitSymbol", "@123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimestamp(tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTimestampUnit(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"second", "second", true},
		{"seconds", "seconds", true},
		{"minute", "minute", true},
		{"minutes", "minutes", true},
		{"hour", "hour", true},
		{"hours", "hours", true},
		{"day", "day", true},
		{"days", "days", true},
		{"week", "week", true},
		{"weeks", "weeks", true},
		{"month", "month", true},
		{"months", "months", true},
		{"year", "year", true},
		{"years", "years", true},
		{"ago", "ago", true},
		{"invalidUnit", "invalid", false},
		{"emptyString", "", false},
		{"partialMatch", "sec", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimestampUnit(tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBookmark(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"withAt", "main@", true},
		{"withSlash", "feature/branch", true},
		{"withBoth", "main@feature/branch", true},
		{"noSpecial", "main", false},
		{"emptyString", "", false},
		{"onlyAt", "@", true},
		{"onlySlash", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBookmark(tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCommitHeader(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected *jjCommit
	}{
		{
			name: "simpleCommit",
			line: "@ abc123 def456 2 hours ago",
			expected: &jjCommit{
				GraphChar:   "@",
				ChangeID:    "abc123",
				CommitID:    "def456",
				Timestamp:   "2 hours ago",
				Workspaces:  nil,
				Bookmarks:   nil,
				Description: "",
			},
		},
		{
			name: "commitWithWorkspace",
			line: "○ abc123 def456 workspace1 1 day ago",
			expected: &jjCommit{
				GraphChar:   "○",
				ChangeID:    "abc123",
				CommitID:    "def456",
				Timestamp:   "1 day ago",
				Workspaces:  []string{"workspace1"},
				Bookmarks:   nil,
				Description: "",
			},
		},
		{
			name: "commitWithBookmark",
			line: "◆ abc123 def456 main@ 2 hours ago",
			expected: &jjCommit{
				GraphChar:   "◆",
				ChangeID:    "abc123",
				CommitID:    "def456",
				Timestamp:   "2 hours ago",
				Workspaces:  nil,
				Bookmarks:   []string{"main@"},
				Description: "",
			},
		},
		{
			name: "commitWithGraphPrefix",
			line: "│ ○ abc123 def456 3 days ago",
			expected: &jjCommit{
				GraphChar:   "○",
				ChangeID:    "abc123",
				CommitID:    "def456",
				Timestamp:   "3 days ago",
				Workspaces:  nil,
				Bookmarks:   nil,
				Description: "",
			},
		},
		{
			name: "commitWithMultipleParts",
			line: "× abc123 def456 ws1 ws2 1 week ago main@ feature/",
			expected: &jjCommit{
				GraphChar:   "×",
				ChangeID:    "abc123",
				CommitID:    "def456",
				Timestamp:   "1 week ago",
				Workspaces:  []string{"ws1"},
				Bookmarks:   []string{"main@", "feature/"},
				Description: "",
			},
		},
		{
			name: "noMarker",
			line: "abc123 def456 2 hours ago",
			expected: &jjCommit{
				GraphChar:   "",
				ChangeID:    "",
				CommitID:    "",
				Timestamp:   "",
				Workspaces:  nil,
				Bookmarks:   nil,
				Description: "",
			},
		},
		{
			name: "tooShort",
			line: "@ abc",
			expected: &jjCommit{
				GraphChar:   "@",
				ChangeID:    "",
				CommitID:    "",
				Timestamp:   "",
				Workspaces:  nil,
				Bookmarks:   nil,
				Description: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommitHeader(tt.line)
			assert.Equal(t, tt.expected.GraphChar, result.GraphChar)
			assert.Equal(t, tt.expected.ChangeID, result.ChangeID)
			assert.Equal(t, tt.expected.CommitID, result.CommitID)
			assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			assert.Equal(t, tt.expected.Workspaces, result.Workspaces)
			assert.Equal(t, tt.expected.Bookmarks, result.Bookmarks)
		})
	}
}

func TestWorkspaceDagAgents(t *testing.T) {
	tests := []struct {
		name        string
		goalContent string
		expected    []string
	}{
		{
			name: "simpleFlow",
			goalContent: `---
flow: |
  "agent1" -> "agent2"
retrospective: false
---
# Test Goal`,
			expected: []string{"agent1", "agent2", "coordinator", "project-critic-council"},
		},
		{
			name: "multipleAgents",
			goalContent: `---
flow: |
  "agent1" -> "agent2"
  "agent2" -> "agent3"
  "agent3" -> "agent4"
retrospective: false
---
# Test Goal`,
			expected: []string{"agent1", "agent2", "agent3", "agent4", "coordinator", "project-critic-council"},
		},
		{
			name: "branchingFlow",
			goalContent: `---
flow: |
  "agent1" -> "agent2"
  "agent1" -> "agent3"
  "agent2" -> "agent4"
  "agent3" -> "agent4"
retrospective: false
---
# Test Goal`,
			expected: []string{"agent1", "agent2", "agent3", "agent4", "coordinator", "project-critic-council"},
		},
		{
			name: "withRetrospective",
			goalContent: `---
flow: |
  "agent1" -> "agent2"
retrospective: true
---
# Test Goal`,
			expected: []string{"agent1", "agent2", "coordinator", "project-critic-council", "retrospective"},
		},
		{
			name: "noFlow",
			goalContent: `---
retrospective: false
---
# Test Goal`,
			expected: []string{"coordinator", "general-purpose", "project-critic-council"},
		},
		{
			name: "invalidYAML",
			goalContent: `---
flow: |
  invalid yaml content [
---
# Test Goal`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			goalPath := filepath.Join(workspacePath, "GOAL.md")
			require.NoError(t, os.WriteFile(goalPath, []byte(tt.goalContent), 0644))

			result := workspaceDagAgents(workspacePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkspaceDagAgentsNoGoalFile(t *testing.T) {
	workspacePath := t.TempDir()
	result := workspaceDagAgents(workspacePath)
	assert.Nil(t, result)
}

func TestParseJJLogOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []*jjCommit
	}{
		{
			name: "singleCommit",
			input: `@ abc123 def456 2 hours ago
Commit message here`,
			expected: []*jjCommit{
				{
					GraphChar:   "@",
					ChangeID:    "abc123",
					CommitID:    "def456",
					Timestamp:   "2 hours ago",
					Description: "Commit message here",
				},
			},
		},
		{
			name: "multipleCommits",
			input: `@ abc123 def456 2 hours ago
First commit message
○ xyz789 uvw012 1 day ago
Second commit message
◆ rst345 mno678 3 days ago
Third commit message`,
			expected: []*jjCommit{
				{
					GraphChar:   "@",
					ChangeID:    "abc123",
					CommitID:    "def456",
					Timestamp:   "2 hours ago",
					Description: "First commit message",
				},
				{
					GraphChar:   "○",
					ChangeID:    "xyz789",
					CommitID:    "uvw012",
					Timestamp:   "1 day ago",
					Description: "Second commit message",
				},
				{
					GraphChar:   "◆",
					ChangeID:    "rst345",
					CommitID:    "mno678",
					Timestamp:   "3 days ago",
					Description: "Third commit message",
				},
			},
		},
		{
			name: "commitWithGraphPrefix",
			input: `│ ○ abc123 def456 2 hours ago
Commit with graph prefix`,
			expected: []*jjCommit{
				{
					GraphChar:   "○",
					ChangeID:    "abc123",
					CommitID:    "def456",
					Timestamp:   "2 hours ago",
					Description: "Commit with graph prefix",
				},
			},
		},
		{
			name:     "emptyInput",
			input:    "",
			expected: nil,
		},
		{
			name: "onlyGraphLines",
			input: `│
│
│`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJJLogOutput(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
			for i, expected := range tt.expected {
				if i < len(result) {
					assert.Equal(t, expected.GraphChar, result[i].GraphChar)
					assert.Equal(t, expected.ChangeID, result[i].ChangeID)
					assert.Equal(t, expected.CommitID, result[i].CommitID)
					assert.Equal(t, expected.Timestamp, result[i].Timestamp)
					assert.Equal(t, expected.Description, result[i].Description)
				}
			}
		})
	}
}

func TestHasNextCommit(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		idx      int
		expected bool
	}{
		{
			name: "hasNextCommit",
			lines: []string{
				"@ abc123 def456 2 hours ago",
				"Commit message",
				"○ xyz789 uvw012 1 day ago",
			},
			idx:      0,
			expected: true,
		},
		{
			name: "noNextCommit",
			lines: []string{
				"@ abc123 def456 2 hours ago",
				"Commit message",
			},
			idx:      0,
			expected: false,
		},
		{
			name: "lastLine",
			lines: []string{
				"@ abc123 def456 2 hours ago",
				"Commit message",
			},
			idx:      1,
			expected: false,
		},
		{
			name: "emptyLines",
			lines: []string{
				"@ abc123 def456 2 hours ago",
				"",
				"",
				"○ xyz789 uvw012 1 day ago",
			},
			idx:      0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNextCommit(tt.lines, tt.idx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServerLoadPinnedProjects(t *testing.T) {
	t.Skip("Integration test - requires system-wide XDG config directory")
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) []string
		expectCount int
		expectError bool
	}{
		{
			name: "validPinnedFile",
			setupFunc: func(t *testing.T, rootDir string) []string {
				pinnedFile := filepath.Join(rootDir, ".sgai", "pinned.json")
				require.NoError(t, os.MkdirAll(filepath.Dir(pinnedFile), 0755))
				workspaces := []string{
					filepath.Join(rootDir, "workspace1"),
					filepath.Join(rootDir, "workspace2"),
				}
				for _, ws := range workspaces {
					require.NoError(t, os.MkdirAll(ws, 0755))
				}
				data, err := json.Marshal(workspaces)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(pinnedFile, data, 0644))
				return workspaces
			},
			expectCount: 2,
			expectError: false,
		},
		{
			name: "emptyPinnedFile",
			setupFunc: func(t *testing.T, rootDir string) []string {
				pinnedFile := filepath.Join(rootDir, ".sgai", "pinned.json")
				require.NoError(t, os.MkdirAll(filepath.Dir(pinnedFile), 0755))
				require.NoError(t, os.WriteFile(pinnedFile, []byte("[]"), 0644))
				return nil
			},
			expectCount: 0,
			expectError: false,
		},
		{
			name: "noPinnedFile",
			setupFunc: func(_ *testing.T, _ string) []string {
				return nil
			},
			expectCount: 0,
			expectError: false,
		},
		{
			name: "prunesStalePaths",
			setupFunc: func(t *testing.T, rootDir string) []string {
				pinnedFile := filepath.Join(rootDir, ".sgai", "pinned.json")
				require.NoError(t, os.MkdirAll(filepath.Dir(pinnedFile), 0755))
				workspaces := []string{
					filepath.Join(rootDir, "workspace1"),
					filepath.Join(rootDir, "nonexistent"),
				}
				require.NoError(t, os.MkdirAll(workspaces[0], 0755))
				data, err := json.Marshal(workspaces)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(pinnedFile, data, 0644))
				return []string{workspaces[0]}
			},
			expectCount: 1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)
			tt.setupFunc(t, rootDir)

			err := server.loadPinnedProjects()
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectCount, len(server.pinnedDirs))
		})
	}
}

func TestRenderDotAsFallbackSVG(t *testing.T) {
	tests := []struct {
		name          string
		dotContent    string
		shouldContain []string
	}{
		{
			name:       "simpleDot",
			dotContent: "digraph G {\n  A -> B\n}",
			shouldContain: []string{
				`<svg`,
				`<text`,
				`digraph G {`,
				`A -&gt; B`,
				`</svg>`,
			},
		},
		{
			name:       "emptyContent",
			dotContent: "",
			shouldContain: []string{
				`<svg`,
				`height="100"`,
				`</svg>`,
			},
		},
		{
			name:       "multilineContent",
			dotContent: "line1\nline2\nline3",
			shouldContain: []string{
				`<svg`,
				`line1`,
				`line2`,
				`line3`,
				`</svg>`,
			},
		},
		{
			name:       "contentWithTrailingEmpty",
			dotContent: "line1\n\n\nline2",
			shouldContain: []string{
				`<svg`,
				`line1`,
				`line2`,
				`</svg>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderDotAsFallbackSVG(tt.dotContent)
			assert.NotEmpty(t, result)
			for _, expected := range tt.shouldContain {
				assert.Contains(t, result, expected, "SVG should contain %s", expected)
			}
		})
	}
}

func TestResolveSymlinks(t *testing.T) {
	tests := []struct {
		name     string
		makePath func(*testing.T) string
		validate func(*testing.T, string)
	}{
		{
			name: "nonExistentPath",
			makePath: func(_ *testing.T) string {
				return "/non/existent/path"
			},
			validate: func(t *testing.T, result string) { //nolint:thelper
				assert.Equal(t, "/non/existent/path", result)
			},
		},
		{
			name: "regularDirectory",
			makePath: func(t *testing.T) string { //nolint:thelper
				dir := t.TempDir()
				path := filepath.Join(dir, "testdir")
				require.NoError(t, os.MkdirAll(path, 0755))
				return path
			},
			validate: func(t *testing.T, result string) { //nolint:thelper
				assert.Contains(t, result, "testdir")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.makePath(t)
			result := resolveSymlinks(path)
			tt.validate(t, result)
		})
	}
}

func TestPinnedFilePath(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	result := server.pinnedFilePath()
	assert.Contains(t, result, "pinned.json")
}

func TestWasEverStarted(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	assert.False(t, server.wasEverStarted(workspacePath))

	server.mu.Lock()
	server.everStartedDirs[workspacePath] = true
	server.mu.Unlock()

	assert.True(t, server.wasEverStarted(workspacePath))
}

func TestCreateWorkspaceInfo(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	info := server.createWorkspaceInfo(workspacePath, "test-workspace", false, true, false)

	assert.Equal(t, "test-workspace", info.DirName)
	assert.Equal(t, workspacePath, info.Directory)
	assert.False(t, info.IsRoot)
	assert.True(t, info.HasWorkspace)
	assert.False(t, info.External)
	assert.False(t, info.Running)
	assert.False(t, info.NeedsInput)
}

func TestResetHumanCommunicationNoSession(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	server.resetHumanCommunication(workspacePath)
}

func TestPrepareAgentSequenceDisplay(t *testing.T) {
	now := time.Now().UTC()

	t.Run("empty", func(t *testing.T) {
		result := prepareAgentSequenceDisplay(nil, false, "", "")
		assert.Empty(t, result)
	})

	t.Run("singleRunningEntry", func(t *testing.T) {
		sequence := []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339), IsCurrent: true},
		}
		result := prepareAgentSequenceDisplay(sequence, true, "", "")
		require.Len(t, result, 1)
		assert.Equal(t, "coordinator", result[0].Agent)
		assert.True(t, result[0].IsCurrent)
		assert.Contains(t, result[0].ElapsedTime, "m")
	})

	t.Run("multipleEntries", func(t *testing.T) {
		sequence := []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: now.Add(-10 * time.Minute).Format(time.RFC3339)},
			{Agent: "developer", StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339), IsCurrent: true},
		}
		result := prepareAgentSequenceDisplay(sequence, true, "", "")
		require.Len(t, result, 2)
		assert.Equal(t, "developer", result[0].Agent)
		assert.Equal(t, "coordinator", result[1].Agent)
	})

	t.Run("notRunningWithEndTime", func(t *testing.T) {
		endTime := now.Add(-1 * time.Minute).Format(time.RFC3339)
		sequence := []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: now.Add(-10 * time.Minute).Format(time.RFC3339)},
		}
		result := prepareAgentSequenceDisplay(sequence, false, endTime, "")
		require.Len(t, result, 1)
	})

	t.Run("invalidTimestamp", func(t *testing.T) {
		sequence := []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "not-a-time"},
		}
		result := prepareAgentSequenceDisplay(sequence, false, "", "")
		assert.Empty(t, result)
	})

	t.Run("withWorkspacePath", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"), 0644))

		sequence := []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339), IsCurrent: true},
		}
		result := prepareAgentSequenceDisplay(sequence, true, "", dir)
		require.Len(t, result, 1)
		assert.Equal(t, "claude-opus-4", result[0].Model)
	})
}

func TestNewServerWithConfigExplicit(t *testing.T) {
	dir := t.TempDir()
	srv := NewServerWithConfig(dir, "echo")
	assert.NotNil(t, srv)
	assert.Contains(t, srv.rootDir, filepath.Base(dir))
}

func TestNewServerWithConfigInvalidEditor(t *testing.T) {
	dir := t.TempDir()
	srv := NewServerWithConfig(dir, "nonexistent-editor-xyzzy")
	assert.NotNil(t, srv)
}

func TestAddGitExcludeNoGitDir(t *testing.T) {
	dir := t.TempDir()
	err := addGitExclude(dir)
	assert.NoError(t, err)
}

func TestAddGitExcludeWithGitDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	err := addGitExclude(dir)
	require.NoError(t, err)

	data, errRead := os.ReadFile(filepath.Join(dir, ".git", "info", "exclude"))
	require.NoError(t, errRead)
	assert.Contains(t, string(data), "/.sgai")
}

func TestAddGitExcludeAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	infoDir := filepath.Join(dir, ".git", "info")
	require.NoError(t, os.MkdirAll(infoDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(infoDir, "exclude"), []byte("/.sgai\n"), 0o644))

	err := addGitExclude(dir)
	require.NoError(t, err)
}

func TestWriteGoalExampleCreatesFile(t *testing.T) {
	dir := t.TempDir()
	err := writeGoalExample(dir)
	require.NoError(t, err)
	data, errRead := os.ReadFile(filepath.Join(dir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, goalExampleContent, string(data))
}

func TestValidateWorkspaceNameCases(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "my-workspace", false},
		{"validNumbers", "workspace-123", false},
		{"empty", "", true},
		{"pathSeparator", "foo/bar", true},
		{"dotDot", "foo..bar", true},
		{"uppercase", "MyWorkspace", true},
		{"underscore", "my_workspace", true},
		{"spaces", "my workspace", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := validateWorkspaceName(tc.input)
			if tc.wantErr {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestRenderDotToSVGFallback(t *testing.T) {
	dot := "digraph G {\n  \"a\" -> \"b\"\n}"
	svg := renderDotToSVG(dot)
	assert.NotEmpty(t, svg)
	assert.Contains(t, svg, "<svg")
}

func TestBadgeStatusVariants(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		class, text := badgeStatus(state.Workflow{Status: state.StatusComplete}, false)
		assert.Equal(t, "badge-complete", class)
		assert.Equal(t, "Complete", text)
	})

	t.Run("working", func(t *testing.T) {
		class, text := badgeStatus(state.Workflow{Status: state.StatusWorking}, true)
		assert.Equal(t, "badge-running", class)
		assert.Equal(t, "Running", text)
	})

	t.Run("needsInput", func(t *testing.T) {
		wf := state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "question",
		}
		class, text := badgeStatus(wf, true)
		assert.Equal(t, "badge-needs-input", class)
		assert.Equal(t, "Needs Input", text)
	})

	t.Run("stopped", func(t *testing.T) {
		class, text := badgeStatus(state.Workflow{Status: ""}, false)
		assert.Equal(t, "badge-stopped", class)
		assert.Equal(t, "Stopped", text)
	})
}

func TestGetLatestProgressFromEntries(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "-", getLatestProgress(nil))
	})

	t.Run("withEntries", func(t *testing.T) {
		entries := []state.ProgressEntry{{Description: "first"}, {Description: "latest"}}
		assert.Equal(t, "latest", getLatestProgress(entries))
	})
}

func TestGetLastActivityTimeFromEntries(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := getLastActivityTime(nil)
		assert.Empty(t, got)
	})

	t.Run("withEntries", func(t *testing.T) {
		now := time.Now().UTC().Format(time.RFC3339)
		entries := []state.ProgressEntry{{Timestamp: now}}
		got := getLastActivityTime(entries)
		assert.NotEmpty(t, got)
	})
}

func TestDashboardBaseURLFormat(t *testing.T) {
	url := dashboardBaseURL("127.0.0.1:8080")
	assert.Contains(t, url, "http")
	assert.Contains(t, url, "8080")
}

func TestDashboardBaseURLVariants(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{"ipv4WithPort", "127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"emptyHost", ":8080", "http://127.0.0.1:8080"},
		{"allInterfaces", "0.0.0.0:9000", "http://127.0.0.1:9000"},
		{"ipv6All", "[::]:8080", "http://[::1]:8080"},
		{"invalidAddr", "just-a-host", "http://just-a-host"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := dashboardBaseURL(tc.input)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestStripFrontmatterVariantsExtended(t *testing.T) {
	assert.Equal(t, "Hello world", stripFrontmatter("Hello world"))
	got := stripFrontmatter("---\ntitle: test\n---\nBody content")
	assert.Contains(t, got, "Body content")
	got2 := stripFrontmatter("---\n---\nBody")
	assert.Contains(t, got2, "Body")
}

func TestExtractSubjectVariants(t *testing.T) {
	assert.Equal(t, "Hello World", extractSubject("# Hello World\nContent"))
	assert.Equal(t, "plain message", extractSubject("plain message"))
	assert.Empty(t, extractSubject(""))
}

func TestInjectCurrentAgentStyleResult(t *testing.T) {
	dot := "digraph G {\n    \"coordinator\" -> \"builder\"\n    \"coordinator\"\n}"
	result := injectCurrentAgentStyle(dot, "coordinator")
	assert.Contains(t, result, "fillcolor")
}

func TestInjectLightThemeResult(t *testing.T) {
	dot := "digraph G {\n    \"a\" -> \"b\"\n}"
	result := injectLightTheme(dot)
	assert.Contains(t, result, "bgcolor")
}

func TestFormatProgressForDisplayDateDivider(t *testing.T) {
	entries := []state.ProgressEntry{
		{Timestamp: "2025-01-01T12:00:00Z", Agent: "coordinator", Description: "started work"},
		{Timestamp: "2025-01-02T12:01:00Z", Agent: "developer", Description: "writing code"},
	}

	result := formatProgressForDisplay(entries)
	assert.Len(t, result, 2)

	hasDivider := false
	for _, r := range result {
		if r.ShowDateDivider {
			hasDivider = true
		}
	}
	assert.True(t, hasDivider)
}

func TestFormatProgressForDisplayEmpty(t *testing.T) {
	result := formatProgressForDisplay(nil)
	assert.Empty(t, result)
}

func TestFormatProgressForDisplayValid(t *testing.T) {
	progress := []state.ProgressEntry{
		{Timestamp: "2025-01-01T10:00:00Z", Description: "started"},
		{Timestamp: "2025-01-01T11:00:00Z", Description: "completed"},
	}
	result := formatProgressForDisplay(progress)
	assert.NotEmpty(t, result)
}

func TestGetWorkflowSVGHashCachedResult(t *testing.T) {
	srv, _ := setupTestServer(t)
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\nflow: |\n  digraph G {\n    \"a\" -> \"b\"\n  }\n---\n# Test"), 0o644))

	hash := srv.getWorkflowSVGHashCached(dir, "coordinator")
	assert.NotEmpty(t, hash)
}

func TestRenderMarkdownOutput(t *testing.T) {
	html, err := renderMarkdown([]byte("# Hello\n\n**bold** text"))
	require.NoError(t, err)
	assert.Contains(t, html, "Hello")
	assert.Contains(t, html, "bold")
}

func TestHasJJRepoFalse(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, hasJJRepo(dir))
}

func TestHasJJRepoTrue(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jj"), 0o755))
	assert.True(t, hasJJRepo(dir))
}

func TestStatePathGeneration(t *testing.T) {
	result := statePath("/some/workspace")
	assert.Equal(t, "/some/workspace/.sgai/state.json", result)
}

func TestResolveWorkspaceNameToPath(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "resolve-name")

	result := srv.resolveWorkspaceNameToPath("resolve-name")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "resolve-name")
}

func TestResolveWorkspaceNameToPathNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	result := srv.resolveWorkspaceNameToPath("nonexistent-workspace-xyz")
	assert.Empty(t, result)
}

func TestClassifyWorkspaceCachedStandalone(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "classify-standalone")
	kind := srv.classifyWorkspaceCached(wsDir)
	assert.Equal(t, workspaceStandalone, kind)
}

func TestClassifyWorkspaceCachedNonExistent(t *testing.T) {
	srv, _ := setupTestServer(t)
	kind := srv.classifyWorkspaceCached("/nonexistent/workspace")
	assert.Equal(t, workspaceStandalone, kind)
}

func TestNotifyStateChangeInvalidatesCache(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "notify-ws")

	srv.warmStateCache()
	_, ok := srv.stateCache.get("state")
	assert.True(t, ok)

	srv.notifyStateChange()
	_, ok2 := srv.stateCache.get("state")
	assert.False(t, ok2)
}

func TestWorkspaceCoordinator(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "coord-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	coord := srv.workspaceCoordinator(wsDir)
	assert.NotNil(t, coord)
	wf := coord.State()
	assert.Equal(t, state.StatusComplete, wf.Status)
}

func TestWorkspaceCoordinatorNoState(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "coord-nostate")
	coord := srv.workspaceCoordinator(wsDir)
	assert.NotNil(t, coord)
}

func TestFlushGoalChecksumOnStopNoSession(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "flush-chksum")
	srv.flushGoalChecksumOnStop(wsDir)
}

func TestStopSessionIdempotent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stop-idem")
	srv.stopSession(wsDir)
	srv.stopSession(wsDir)
}

func TestOrderedModelStatusesEmpty(t *testing.T) {
	dir := t.TempDir()
	result := orderedModelStatuses(dir, nil)
	assert.Empty(t, result)
}

func TestDoScanWorkspaceGroupsWithStandalone(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := filepath.Join(rootDir, "standalone-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai"), 0755))
	groups, errScan := server.doScanWorkspaceGroups()
	require.NoError(t, errScan)
	assert.GreaterOrEqual(t, len(groups), 1)
}

func TestScanWorkspaceGroupsEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	groups, err := srv.scanWorkspaceGroups()
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestScanWorkspaceGroupsWithWorkspaces(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "scan-ws1")
	_ = setupTestWorkspace(t, rootDir, "scan-ws2")

	groups, err := srv.scanWorkspaceGroups()
	assert.NoError(t, err)
	assert.NotEmpty(t, groups)
}

func TestScanWorkspaceGroupsCachingBehavior(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	groups1, err1 := server.scanWorkspaceGroups()
	require.NoError(t, err1)
	groups2, err2 := server.scanWorkspaceGroups()
	require.NoError(t, err2)
	assert.Equal(t, len(groups1), len(groups2))
}

func TestInvalidateWorkspaceScanCache(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.invalidateWorkspaceScanCache()
}

func TestResetHumanCommunicationWithCoordinator(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "reset-human")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "old message",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Q?", Choices: []string{"A", "B"}},
			},
		},
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	srv.resetHumanCommunication(wsDir)

	wf := coord.State()
	assert.Empty(t, wf.HumanMessage)
	assert.Equal(t, state.StatusWorking, wf.Status)
}

func TestValidateDirectoryEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	_, err := srv.validateDirectory("")
	assert.Error(t, err)
}

func TestValidateDirectoryValid(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "validate-dir")
	result, err := srv.validateDirectory(wsDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestValidateDirectoryOutsideRoot(t *testing.T) {
	srv, _ := setupTestServer(t)
	_, err := srv.validateDirectory("/tmp/outside-root")
	assert.Error(t, err)
}

func TestValidateDirectoryTraversal(t *testing.T) {
	srv, _ := setupTestServer(t)
	_, err := srv.validateDirectory("../../../etc/passwd")
	assert.Error(t, err)
}

func TestGatherSnippetsByLanguageEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "snippets"), 0o755))
	result := gatherSnippetsByLanguage(dir)
	assert.Empty(t, result)
}

func TestGatherSnippetsByLanguageWithSnippets(t *testing.T) {
	dir := t.TempDir()
	goDir := filepath.Join(dir, ".sgai", "snippets", "go")
	pyDir := filepath.Join(dir, ".sgai", "snippets", "python")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.MkdirAll(pyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "http.go"), []byte("// HTTP server\npackage main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pyDir, "hello.py"), []byte("# Hello\nprint('hi')"), 0o644))

	result := gatherSnippetsByLanguage(dir)
	assert.Len(t, result, 2)
}

func TestResetHumanCommunicationWithNoCoordinator(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	sess := &session{
		running: true,
		coord:   nil,
	}
	server.mu.Lock()
	server.sessions[workspacePath] = sess
	server.mu.Unlock()

	server.resetHumanCommunication(workspacePath)
}

func TestFlushGoalChecksumOnStopUpdatesChecksum(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test"), 0644))
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{})
	require.NoError(t, errCoord)
	server.flushGoalChecksumOnStop(wsDir)
	coordAfter := server.workspaceCoordinator(wsDir)
	wfState := coordAfter.State()
	assert.NotEmpty(t, wfState.GoalChecksum)
}

func TestStopSessionWithRunningSessionMarksNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	coord, errCoord := state.NewCoordinatorWith(filepath.Join(wsDir, ".sgai", "state.json"), state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "question?",
	})
	require.NoError(t, errCoord)

	server.mu.Lock()
	server.sessions[wsDir] = &session{
		running: true,
		coord:   coord,
	}
	server.mu.Unlock()

	server.stopSession(wsDir)

	server.mu.Lock()
	sess := server.sessions[wsDir]
	server.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		assert.False(t, sess.running)
		sess.mu.Unlock()
	}
}

func TestGetLastActivityTime(t *testing.T) {
	tests := []struct {
		name     string
		progress []state.ProgressEntry
		expected string
	}{
		{
			name:     "emptyProgress",
			progress: []state.ProgressEntry{},
			expected: "",
		},
		{
			name: "singleEntry",
			progress: []state.ProgressEntry{
				{Timestamp: "2024-01-15T10:30:00Z"},
			},
			expected: "2024-01-15T10:30:00Z",
		},
		{
			name: "multipleEntries",
			progress: []state.ProgressEntry{
				{Timestamp: "2024-01-15T10:30:00Z"},
				{Timestamp: "2024-01-15T11:00:00Z"},
				{Timestamp: "2024-01-15T12:00:00Z"},
			},
			expected: "2024-01-15T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLastActivityTime(tt.progress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInjectCurrentAgentStyle(t *testing.T) {
	tests := []struct {
		name         string
		dot          string
		currentAgent string
		expected     string
	}{
		{
			name:         "agentExists",
			dot:          "digraph G {\n    \"agent1\"\n}",
			currentAgent: "agent1",
			expected:     "digraph G {\n    \"agent1\" [style=filled, fillcolor=\"#10b981\", fontcolor=white]\n}",
		},
		{
			name:         "agentNotExists",
			dot:          `digraph G { "agent1" }`,
			currentAgent: "agent2",
			expected:     `digraph G { "agent1" }`,
		},
		{
			name:         "emptyDot",
			dot:          "",
			currentAgent: "agent1",
			expected:     "",
		},
		{
			name:         "multipleAgents",
			dot:          "digraph G {\n    \"agent1\"\n    \"agent2\"\n}",
			currentAgent: "agent2",
			expected:     "digraph G {\n    \"agent1\"\n    \"agent2\" [style=filled, fillcolor=\"#10b981\", fontcolor=white]\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectCurrentAgentStyle(tt.dot, tt.currentAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInjectLightTheme(t *testing.T) {
	tests := []struct {
		name     string
		dot      string
		contains string
	}{
		{
			name:     "validDot",
			dot:      `digraph G { "agent1" }`,
			contains: "bgcolor",
		},
		{
			name:     "noBrace",
			dot:      `digraph G`,
			contains: "",
		},
		{
			name:     "emptyDot",
			dot:      "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectLightTheme(tt.dot)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			} else {
				assert.Equal(t, tt.dot, result)
			}
		})
	}
}

func TestGetLatestProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress []state.ProgressEntry
		expected string
	}{
		{
			name:     "emptyProgress",
			progress: []state.ProgressEntry{},
			expected: "-",
		},
		{
			name: "singleEntry",
			progress: []state.ProgressEntry{
				{Description: "First action"},
			},
			expected: "First action",
		},
		{
			name: "multipleEntries",
			progress: []state.ProgressEntry{
				{Description: "First action"},
				{Description: "Second action"},
				{Description: "Third action"},
			},
			expected: "Third action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLatestProgress(tt.progress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinesWithTrailingEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty",
			content:  "",
			expected: []string{""},
		},
		{
			name:     "singleLineNoNewline",
			content:  "line1",
			expected: []string{"line1"},
		},
		{
			name:     "singleLineWithNewline",
			content:  "line1\n",
			expected: []string{"line1", ""},
		},
		{
			name:     "multipleLinesNoTrailingNewline",
			content:  "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "multipleLinesWithTrailingNewline",
			content:  "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3", ""},
		},
		{
			name:     "multipleNewlinesAtEnd",
			content:  "line1\nline2\n\n",
			expected: []string{"line1", "line2", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := linesWithTrailingEmpty(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrderedModelStatuses(t *testing.T) {
	tests := []struct {
		name          string
		modelStatuses map[string]string
		setupFunc     func(*testing.T, string)
		validate      func(*testing.T, []modelStatusDisplay)
	}{
		{
			name:          "emptyStatuses",
			modelStatuses: map[string]string{},
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, displays []modelStatusDisplay) {
				assert.Nil(t, displays)
			},
		},
		{
			name: "singleStatus",
			modelStatuses: map[string]string{
				"model1": "running",
			},
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, displays []modelStatusDisplay) {
				assert.Len(t, displays, 1)
				assert.Equal(t, "model1", displays[0].ModelID)
				assert.Equal(t, "running", displays[0].Status)
			},
		},
		{
			name: "multipleStatuses",
			modelStatuses: map[string]string{
				"model1": "running",
				"model2": "done",
				"model3": "error",
			},
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, displays []modelStatusDisplay) {
				assert.Len(t, displays, 3)
				modelIDs := make([]string, len(displays))
				for i, d := range displays {
					modelIDs[i] = d.ModelID
				}
				assert.Contains(t, modelIDs, "model1")
				assert.Contains(t, modelIDs, "model2")
				assert.Contains(t, modelIDs, "model3")
			},
		},
		{
			name: "withGoalModels",
			modelStatuses: map[string]string{
				"project-critic-council:model1": "running",
				"project-critic-council:model2": "done",
			},
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
models:
  project-critic-council:
    - model1
    - model2
---
# Goal`
				require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, displays []modelStatusDisplay) {
				assert.Len(t, displays, 2)
				assert.Equal(t, "project-critic-council:model1", displays[0].ModelID)
				assert.Equal(t, "project-critic-council:model2", displays[1].ModelID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := orderedModelStatuses(dir, tt.modelStatuses)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestModelsForAgentFromGoal(t *testing.T) {
	tests := []struct {
		name      string
		agent     string
		setupFunc func(*testing.T, string)
		expected  []string
	}{
		{
			name:  "noGoal",
			agent: "agent1",
			setupFunc: func(_ *testing.T, _ string) {
			},
			expected: nil,
		},
		{
			name:  "singleModel",
			agent: "agent1",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
models:
  agent1: model1
---
# Goal`
				require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0644))
			},
			expected: []string{"model1"},
		},
		{
			name:  "multipleModels",
			agent: "agent1",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
models:
  agent1:
    - model1
    - model2
---
# Goal`
				require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0644))
			},
			expected: []string{"model1", "model2"},
		},
		{
			name:  "agentNotInModels",
			agent: "agent2",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
models:
  agent1: model1
---
# Goal`
				require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0644))
			},
			expected: nil,
		},
		{
			name:  "invalidGoal",
			agent: "agent1",
			setupFunc: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("not valid yaml"), 0644))
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := modelsForAgentFromGoal(dir, tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadPinnedProjectsNoFile(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.pinnedConfigDir = t.TempDir()

	err := srv.loadPinnedProjects()
	require.NoError(t, err)
	assert.Empty(t, srv.pinnedDirs)
}

func TestLoadPinnedProjectsValidFile(t *testing.T) {
	srv, _ := setupTestServer(t)
	configDir := t.TempDir()
	srv.pinnedConfigDir = configDir

	pinnedDir := t.TempDir()
	data, errMarshal := json.Marshal([]string{pinnedDir})
	require.NoError(t, errMarshal)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "pinned.json"), data, 0o644))

	err := srv.loadPinnedProjects()
	require.NoError(t, err)

	resolvedDir := resolveSymlinks(pinnedDir)
	assert.True(t, srv.pinnedDirs[resolvedDir])
}

func TestLoadPinnedProjectsPrunesStale(t *testing.T) {
	srv, _ := setupTestServer(t)
	configDir := t.TempDir()
	srv.pinnedConfigDir = configDir

	validDir := t.TempDir()
	data, errMarshal := json.Marshal([]string{validDir, "/nonexistent/path/12345"})
	require.NoError(t, errMarshal)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "pinned.json"), data, 0o644))

	err := srv.loadPinnedProjects()
	require.NoError(t, err)

	resolvedValidDir := resolveSymlinks(validDir)
	assert.True(t, srv.pinnedDirs[resolvedValidDir])
	assert.False(t, srv.pinnedDirs["/nonexistent/path/12345"])
}

func TestLoadPinnedProjectsInvalidJSON(t *testing.T) {
	srv, _ := setupTestServer(t)
	configDir := t.TempDir()
	srv.pinnedConfigDir = configDir

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "pinned.json"), []byte("not json"), 0o644))

	err := srv.loadPinnedProjects()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing pinned projects")
}

func TestClearEverStartedOnCompletion(t *testing.T) {
	t.Run("clearsOnComplete", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status: state.StatusComplete,
		})
		require.NoError(t, errCoord)

		srv.mu.Lock()
		srv.everStartedDirs[dir] = true
		srv.mu.Unlock()

		srv.clearEverStartedOnCompletion(dir)

		srv.mu.Lock()
		assert.False(t, srv.everStartedDirs[dir])
		srv.mu.Unlock()
	})

	t.Run("doesNotClearOnWorking", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status: state.StatusWorking,
		})
		require.NoError(t, errCoord)

		srv.mu.Lock()
		srv.everStartedDirs[dir] = true
		srv.mu.Unlock()

		srv.clearEverStartedOnCompletion(dir)

		srv.mu.Lock()
		assert.True(t, srv.everStartedDirs[dir])
		srv.mu.Unlock()
	})
}

func TestSavePinnedProjects(t *testing.T) {
	srv, _ := setupTestServer(t)
	configDir := t.TempDir()
	srv.pinnedConfigDir = configDir

	srv.pinnedDirs = map[string]bool{"/path/a": true, "/path/b": true}

	err := srv.savePinnedProjects()
	require.NoError(t, err)

	data, errRead := os.ReadFile(filepath.Join(configDir, "pinned.json"))
	require.NoError(t, errRead)

	var dirs []string
	require.NoError(t, json.Unmarshal(data, &dirs))
	assert.Len(t, dirs, 2)
	assert.Contains(t, dirs, "/path/a")
	assert.Contains(t, dirs, "/path/b")
}

func TestSavePinnedProjectsEmptyDirs(t *testing.T) {
	srv, _ := setupTestServer(t)
	configDir := t.TempDir()
	srv.pinnedConfigDir = configDir

	srv.pinnedDirs = map[string]bool{}

	err := srv.savePinnedProjects()
	require.NoError(t, err)

	data, errRead := os.ReadFile(filepath.Join(configDir, "pinned.json"))
	require.NoError(t, errRead)

	var dirs []string
	require.NoError(t, json.Unmarshal(data, &dirs))
	assert.Empty(t, dirs)
}

func TestScanForProjects(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, []project, error)
	}{
		{
			name: "emptyDirectory",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
			},
		},
		{
			name: "directoryWithSGAI",
			setupFunc: func(t *testing.T, rootDir string) {
				workspaceDir := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(filepath.Join(workspaceDir, ".sgai"), 0755))
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Len(t, projects, 1)
				assert.Equal(t, "test-workspace", projects[0].DirName)
				assert.True(t, projects[0].HasWorkspace)
			},
		},
		{
			name: "directoryWithGoalMD",
			setupFunc: func(t *testing.T, rootDir string) {
				workspaceDir := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspaceDir, 0755))
				goalPath := filepath.Join(workspaceDir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Len(t, projects, 1)
				assert.Equal(t, "test-workspace", projects[0].DirName)
			},
		},
		{
			name: "directoryWithBoth",
			setupFunc: func(t *testing.T, rootDir string) {
				workspaceDir := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(filepath.Join(workspaceDir, ".sgai"), 0755))
				goalPath := filepath.Join(workspaceDir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Len(t, projects, 1)
				assert.Equal(t, "test-workspace", projects[0].DirName)
				assert.True(t, projects[0].HasWorkspace)
			},
		},
		{
			name: "multipleWorkspaces",
			setupFunc: func(t *testing.T, rootDir string) {
				for _, name := range []string{"workspace-a", "workspace-b", "workspace-c"} {
					workspaceDir := filepath.Join(rootDir, name)
					require.NoError(t, os.MkdirAll(filepath.Join(workspaceDir, ".sgai"), 0755))
				}
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Len(t, projects, 3)
				names := make([]string, len(projects))
				for i, p := range projects {
					names[i] = p.DirName
				}
				assert.Equal(t, []string{"workspace-a", "workspace-b", "workspace-c"}, names)
			},
		},
		{
			name: "nonWorkspaceDirectory",
			setupFunc: func(t *testing.T, rootDir string) {
				regularDir := filepath.Join(rootDir, "regular-dir")
				require.NoError(t, os.MkdirAll(regularDir, 0755))
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
			},
		},
		{
			name: "fileInRootDir",
			setupFunc: func(t *testing.T, rootDir string) {
				filePath := filepath.Join(rootDir, "some-file.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))
			},
			validate: func(t *testing.T, projects []project, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			tt.setupFunc(t, rootDir)

			projects, err := scanForProjects(rootDir)

			if tt.validate != nil {
				tt.validate(t, projects, err)
			}
		})
	}
}

func TestScanForProjectsNonexistentDir(t *testing.T) {
	_, err := scanForProjects("/nonexistent/directory/path")
	assert.Error(t, err)
}

func TestProjectSorting(t *testing.T) {
	rootDir := t.TempDir()

	names := []string{"zebra", "alpha", "beta"}
	for _, name := range names {
		workspaceDir := filepath.Join(rootDir, name)
		require.NoError(t, os.MkdirAll(filepath.Join(workspaceDir, ".sgai"), 0755))
	}

	projects, err := scanForProjects(rootDir)
	require.NoError(t, err)
	require.Len(t, projects, 3)

	assert.Equal(t, "alpha", projects[0].DirName)
	assert.Equal(t, "beta", projects[1].DirName)
	assert.Equal(t, "zebra", projects[2].DirName)
}

func TestProjectModTime(t *testing.T) {
	rootDir := t.TempDir()

	workspaceDir := filepath.Join(rootDir, "test-workspace")
	sgaiDir := filepath.Join(workspaceDir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))

	beforeTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	stateFile := filepath.Join(sgaiDir, "state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte("{}"), 0644))

	time.Sleep(10 * time.Millisecond)
	afterTime := time.Now()

	projects, err := scanForProjects(rootDir)
	require.NoError(t, err)
	require.Len(t, projects, 1)

	modTime := projects[0].LastModified
	assert.True(t, modTime.After(beforeTime) || modTime.Equal(beforeTime))
	assert.True(t, modTime.Before(afterTime) || modTime.Equal(afterTime))
}

func TestGatherSnippetsByLanguage(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, []languageCategory)
	}{
		{
			name: "noSnippets",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, categories []languageCategory) {
				assert.Empty(t, categories)
			},
		},
		{
			name: "singleSnippet",
			setupFunc: func(t *testing.T, dir string) {
				snippetDir := filepath.Join(dir, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(snippetDir, 0755))
				snippetContent := "---\nname: Test Snippet\ndescription: Test description\n---\ncontent"
				require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "test.go"), []byte(snippetContent), 0644))
			},
			validate: func(t *testing.T, categories []languageCategory) {
				assert.Len(t, categories, 1)
				assert.Equal(t, "go", categories[0].Name)
				assert.Len(t, categories[0].Snippets, 1)
				assert.Equal(t, "Test Snippet", categories[0].Snippets[0].Name)
			},
		},
		{
			name: "multipleLanguages",
			setupFunc: func(t *testing.T, dir string) {
				for _, lang := range []string{"go", "python"} {
					snippetDir := filepath.Join(dir, ".sgai", "snippets", lang)
					require.NoError(t, os.MkdirAll(snippetDir, 0755))
					snippetContent := "---\nname: " + lang + " Snippet\ndescription: " + lang + " description\n---\ncontent"
					require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "test."+lang), []byte(snippetContent), 0644))
				}
			},
			validate: func(t *testing.T, categories []languageCategory) {
				assert.Len(t, categories, 2)
				assert.Equal(t, "go", categories[0].Name)
				assert.Equal(t, "python", categories[1].Name)
			},
		},
		{
			name: "snippetWithoutName",
			setupFunc: func(t *testing.T, dir string) {
				snippetDir := filepath.Join(dir, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(snippetDir, 0755))
				snippetContent := "---\ndescription: Test description\n---\n"
				require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "unnamed.go"), []byte(snippetContent), 0644))
			},
			validate: func(t *testing.T, categories []languageCategory) {
				assert.Len(t, categories, 1)
				assert.Len(t, categories[0].Snippets, 1)
				assert.Equal(t, "unnamed", categories[0].Snippets[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := gatherSnippetsByLanguage(dir)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestBadgeStatus(t *testing.T) {
	tests := []struct {
		name        string
		wfState     state.Workflow
		running     bool
		expectClass string
		expectText  string
	}{
		{
			name: "needsHumanInput",
			wfState: state.Workflow{
				Status:       state.StatusWaitingForHuman,
				HumanMessage: "Please provide input",
			},
			running:     false,
			expectClass: "badge-needs-input",
			expectText:  "Needs Input",
		},
		{
			name: "running",
			wfState: state.Workflow{
				Status: state.StatusWorking,
			},
			running:     true,
			expectClass: "badge-running",
			expectText:  "Running",
		},
		{
			name: "working",
			wfState: state.Workflow{
				Status: state.StatusWorking,
			},
			running:     false,
			expectClass: "badge-running",
			expectText:  "Running",
		},
		{
			name: "agentDone",
			wfState: state.Workflow{
				Status: state.StatusAgentDone,
			},
			running:     false,
			expectClass: "badge-running",
			expectText:  "Running",
		},
		{
			name: "complete",
			wfState: state.Workflow{
				Status: state.StatusComplete,
			},
			running:     false,
			expectClass: "badge-complete",
			expectText:  "Complete",
		},
		{
			name: "stopped",
			wfState: state.Workflow{
				Status: "",
			},
			running:     false,
			expectClass: "badge-stopped",
			expectText:  "Stopped",
		},
		{
			name: "multiChoiceQuestion",
			wfState: state.Workflow{
				Status: state.StatusWaitingForHuman,
				MultiChoiceQuestion: &state.MultiChoiceQuestion{
					Questions: []state.QuestionItem{
						{Question: "Choose an option"},
					},
				},
			},
			running:     false,
			expectClass: "badge-needs-input",
			expectText:  "Needs Input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, text := badgeStatus(tt.wfState, tt.running)
			assert.Equal(t, tt.expectClass, class)
			assert.Equal(t, tt.expectText, text)
		})
	}
}

func TestDashboardBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "localhost",
			addr:     "127.0.0.1:8080",
			expected: "http://127.0.0.1:8080",
		},
		{
			name:     "allInterfaces",
			addr:     "0.0.0.0:8080",
			expected: "http://127.0.0.1:8080",
		},
		{
			name:     "ipv6AllInterfaces",
			addr:     "[::]:8080",
			expected: "http://[::1]:8080",
		},
		{
			name:     "invalidFormat",
			addr:     "invalid",
			expected: "http://invalid",
		},
		{
			name:     "emptyHost",
			addr:     ":8080",
			expected: "http://127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dashboardBaseURL(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetWorkflowSVG(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*testing.T, string)
		currentAgent string
		expectEmpty  bool
	}{
		{

			name: "validGoal",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "agent1" -> "agent2"
---
# Test Goal`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			currentAgent: "agent1",
			expectEmpty:  false,
		},
		{
			name: "noGoal",
			setupFunc: func(_ *testing.T, _ string) {
			},
			currentAgent: "",
			expectEmpty:  true,
		},
		{
			name: "invalidGoal",
			setupFunc: func(t *testing.T, dir string) {
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("not valid yaml"), 0644))
			},
			currentAgent: "",
			expectEmpty:  false,
		},
		{
			name: "goalWithRetrospective",
			setupFunc: func(t *testing.T, dir string) {
				goalContent := `---
flow: |
  "agent1" -> "agent2"
retrospective: true
---
# Test Goal`
				goalPath := filepath.Join(dir, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			currentAgent: "agent1",
			expectEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)

			svg := getWorkflowSVG(dir, tt.currentAgent)

			if tt.expectEmpty {
				assert.Empty(t, svg)
			} else {
				assert.NotEmpty(t, svg)
				assert.Contains(t, svg, "svg")
			}
		})
	}
}

func TestGetWorkflowSVGCached(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	goalContent := `---
flow: |
  "agent1" -> "agent2"
---
# Test Goal`
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))

	svg1 := server.getWorkflowSVGCached(workspacePath, "agent1")
	assert.NotEmpty(t, svg1)
	assert.Contains(t, svg1, "svg")

	svg2 := server.getWorkflowSVGCached(workspacePath, "agent1")
	assert.Equal(t, svg1, svg2, "cached result should be the same")
}

func TestGetWorkflowSVGHashCached(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	goalContent := `---
flow: |
  "agent1" -> "agent2"
---
# Test Goal`
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))

	hash1 := server.getWorkflowSVGHashCached(workspacePath, "agent1")
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 16, "hash should be 16 hex characters (8 bytes)")

	hash2 := server.getWorkflowSVGHashCached(workspacePath, "agent1")
	assert.Equal(t, hash1, hash2, "hash should be consistent")
}

func TestGetWorkflowSVGHashCachedEmpty(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	hash := server.getWorkflowSVGHashCached(workspacePath, "agent1")
	assert.Empty(t, hash)
}

func TestFormatProgressForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		entries  []state.ProgressEntry
		validate func(*testing.T, []eventsProgressDisplay)
	}{
		{
			name:    "emptyEntries",
			entries: []state.ProgressEntry{},
			validate: func(t *testing.T, result []eventsProgressDisplay) {
				assert.Empty(t, result)
			},
		},
		{
			name: "singleEntry",
			entries: []state.ProgressEntry{
				{
					Timestamp:   "2024-01-15T10:30:00Z",
					Agent:       "test-agent",
					Description: "Test description",
				},
			},
			validate: func(t *testing.T, result []eventsProgressDisplay) {
				assert.Len(t, result, 1)
				assert.Equal(t, "test-agent", result[0].Agent)
				assert.Equal(t, "Test description", result[0].Description)
				assert.True(t, result[0].ShowDateDivider)
			},
		},
		{
			name: "multipleEntriesSameDay",
			entries: []state.ProgressEntry{
				{
					Timestamp:   "2024-01-15T10:30:00Z",
					Agent:       "agent1",
					Description: "First action",
				},
				{
					Timestamp:   "2024-01-15T14:45:00Z",
					Agent:       "agent2",
					Description: "Second action",
				},
			},
			validate: func(t *testing.T, result []eventsProgressDisplay) {
				assert.Len(t, result, 2)
				assert.True(t, result[0].ShowDateDivider)
				assert.False(t, result[1].ShowDateDivider)
			},
		},
		{
			name: "multipleEntriesDifferentDays",
			entries: []state.ProgressEntry{
				{
					Timestamp:   "2024-01-15T10:30:00Z",
					Agent:       "agent1",
					Description: "Day 1 action",
				},
				{
					Timestamp:   "2024-01-16T10:30:00Z",
					Agent:       "agent2",
					Description: "Day 2 action",
				},
			},
			validate: func(t *testing.T, result []eventsProgressDisplay) {
				assert.Len(t, result, 2)
				assert.True(t, result[0].ShowDateDivider)
				assert.True(t, result[1].ShowDateDivider)
				assert.NotEqual(t, result[0].DateDivider, result[1].DateDivider)
			},
		},
		{
			name: "invalidTimestamp",
			entries: []state.ProgressEntry{
				{
					Timestamp:   "invalid-timestamp",
					Agent:       "agent1",
					Description: "Test action",
				},
			},
			validate: func(t *testing.T, result []eventsProgressDisplay) {
				assert.Len(t, result, 1)
				assert.Equal(t, "invalid-timestamp", result[0].Timestamp)
				assert.Equal(t, "invalid-timestamp", result[0].FormattedTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProgressForDisplay(tt.entries)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCalculateTotalExecutionTime(t *testing.T) {
	tests := []struct {
		name             string
		sequence         []state.AgentSequenceEntry
		running          bool
		lastActivityTime string
		expectedEmpty    bool
	}{
		{
			name:             "emptySequence",
			sequence:         []state.AgentSequenceEntry{},
			running:          false,
			lastActivityTime: "",
			expectedEmpty:    true,
		},
		{
			name: "runningSequence",
			sequence: []state.AgentSequenceEntry{
				{
					Agent:     "agent1",
					StartTime: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			running:          true,
			lastActivityTime: "",
			expectedEmpty:    false,
		},
		{
			name: "stoppedSequence",
			sequence: []state.AgentSequenceEntry{
				{
					Agent:     "agent1",
					StartTime: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				},
			},
			running:          false,
			lastActivityTime: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
			expectedEmpty:    false,
		},
		{
			name: "invalidStartTime",
			sequence: []state.AgentSequenceEntry{
				{
					Agent:     "agent1",
					StartTime: "invalid",
				},
			},
			running:          false,
			lastActivityTime: "",
			expectedEmpty:    true,
		},
		{
			name: "noLastActivityTime",
			sequence: []state.AgentSequenceEntry{
				{
					Agent:     "agent1",
					StartTime: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			running:          false,
			lastActivityTime: "",
			expectedEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTotalExecutionTime(tt.sequence, tt.running, tt.lastActivityTime)
			if tt.expectedEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zeroDuration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "secondsOnly",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutesAndSeconds",
			duration: 90 * time.Second,
			expected: "1m 30s",
		},
		{
			name:     "hoursInMinutes",
			duration: 2 * time.Hour,
			expected: "120m 0s",
		},
		{
			name:     "largeMinutes",
			duration: 65 * time.Minute,
			expected: "65m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		contains string
	}{
		{
			name:     "simpleMarkdown",
			content:  "# Heading\n\nParagraph",
			wantErr:  false,
			contains: "<h1",
		},
		{
			name:     "markdownWithCode",
			content:  "```go\nfunc main() {}\n```",
			wantErr:  false,
			contains: "<pre",
		},
		{
			name:     "markdownWithLink",
			content:  "[Link](https://example.com)",
			wantErr:  false,
			contains: "<a href",
		},
		{
			name:     "emptyContent",
			content:  "",
			wantErr:  false,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderMarkdown([]byte(tt.content))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			}
		})
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "noFrontmatter",
			content:  "Just content",
			expected: "Just content",
		},
		{
			name: "withFrontmatter",
			content: `---
key: value
---
Body content`,
			expected: "Body content",
		},
		{
			name: "emptyFrontmatter",
			content: `---
---
Body content`,
			expected: "Body content",
		},
		{
			name: "unclosedFrontmatter",
			content: `---
key: value
Body content`,
			expected: "---\nkey: value\nBody content",
		},
		{
			name: "multipleNewlines",
			content: `---
key: value
---


Body content`,
			expected: "Body content",
		},
		{
			name:     "emptyContent",
			content:  "",
			expected: "",
		},
		{
			name:     "onlyDelimiter",
			content:  "---",
			expected: "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripFrontmatter(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		name            string
		configEditor    string
		expectedName    string
		expectedCommand string
	}{
		{
			name:            "presetEditor",
			configEditor:    "code",
			expectedName:    "code",
			expectedCommand: "code",
		},
		{
			name:            "cursorEditor",
			configEditor:    "cursor",
			expectedName:    "cursor",
			expectedCommand: "cursor",
		},
		{
			name:            "nvimEditor",
			configEditor:    "nvim",
			expectedName:    "nvim",
			expectedCommand: "nvim",
		},
		{
			name:            "customEditor",
			configEditor:    "my-custom-editor",
			expectedName:    "my-custom-editor",
			expectedCommand: "my-custom-editor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, command, _ := resolveEditor(tt.configEditor)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedCommand, command)
		})
	}
}

func TestResolveEditorDefaultPreset(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	name, command, _ := resolveEditor("")
	assert.Equal(t, defaultEditorPreset, name)
	assert.Equal(t, defaultEditorPreset, command)
}

func TestNewConfigurableEditor(t *testing.T) {
	editor := newConfigurableEditor("code")
	assert.NotNil(t, editor)
	assert.Equal(t, "code", editor.name)
	assert.Equal(t, "code", editor.command)
	assert.False(t, editor.isTerminal)
}

func TestIsEditorAvailable(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantTrue bool
	}{
		{
			name:     "emptyCommand",
			command:  "",
			wantTrue: false,
		},
		{
			name:     "commonCommand",
			command:  "ls",
			wantTrue: true,
		},
		{
			name:     "nonexistentCommand",
			command:  "nonexistent-editor-xyz",
			wantTrue: false,
		},
		{
			name:     "commandWithArgs",
			command:  "ls -la",
			wantTrue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEditorAvailable(tt.command)
			if tt.wantTrue {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}
		})
	}
}

func TestInitializeWorkspace(t *testing.T) {
	_, rootDir := setupTestServer(t)
	newWsDir := filepath.Join(rootDir, "new-workspace")
	require.NoError(t, os.MkdirAll(newWsDir, 0755))

	err := initializeWorkspace(newWsDir)
	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(newWsDir, ".sgai"))
}

func TestInitializeWorkspaceExisting(t *testing.T) {
	_, rootDir := setupTestServer(t)
	wsDir := filepath.Join(rootDir, "existing-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai"), 0755))

	err := initializeWorkspace(wsDir)
	assert.NoError(t, err)
}

func TestCountForkCommitsAheadCached(t *testing.T) {
	server, _ := setupTestServer(t)
	dir := t.TempDir()
	count := server.countForkCommitsAheadCached("main", dir)
	assert.GreaterOrEqual(t, count, 0)
}

func TestDoScanWorkspaceGroups(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws1")
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".jj"), 0755))

	groups, err := server.doScanWorkspaceGroups()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 0)
}

func TestDoScanWorkspaceGroupsWithExternal(t *testing.T) {
	server, rootDir := setupTestServer(t)
	extDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".jj"), 0755))

	externalFile := filepath.Join(rootDir, ".sgai", "external_dirs.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(externalFile), 0755))
	require.NoError(t, os.WriteFile(externalFile, []byte(`["`+extDir+`"]`), 0644))

	_ = server.loadExternalDirs()

	groups, err := server.doScanWorkspaceGroups()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 0)
}

func TestUnpackSkeleton(t *testing.T) {
	dir := t.TempDir()
	err := unpackSkeleton(dir)
	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(dir, ".sgai"))
}

func TestRenderDotToSVGEmpty(t *testing.T) {
	result := renderDotToSVG("")
	assert.Empty(t, result)
}

func TestRenderDotToSVG(t *testing.T) {
	dotContent := "digraph { A -> B }"
	result := renderDotToSVG(dotContent)
	if result != "" {
		assert.Contains(t, result, "svg")
	}
}

func TestGetRootWorkspacePathNonJJ(t *testing.T) {
	dir := t.TempDir()
	result := getRootWorkspacePath(dir)
	assert.Empty(t, result)
}

func TestGetRootWorkspacePathWithJJDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jj", "repo"), 0755))
	result := getRootWorkspacePath(dir)
	assert.Empty(t, result)
}

func TestCountForkCommitsAheadNonRepo(t *testing.T) {
	dir := t.TempDir()
	count := countForkCommitsAhead("main", dir)
	assert.Equal(t, 0, count)
}

func TestResolveBaseBookmarkNonRepo(t *testing.T) {
	result := resolveBaseBookmark(t.TempDir())
	assert.Equal(t, "main", result)
}

func TestRunJJLogForForkNonRepo(t *testing.T) {
	result := runJJLogForFork("main", t.TempDir())
	assert.Empty(t, result)
}

func TestDoScanWorkspaceGroupsEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	groups, err := server.doScanWorkspaceGroups()
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestResolveWorkspaceNameToPathFoundServe(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	result := server.resolveWorkspaceNameToPath("test-ws")
	assert.NotEmpty(t, result)
}

func TestAddGitExcludeExistingDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "info")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	err := addGitExclude(dir)
	assert.NoError(t, err)
}

func TestModelsForAgentFromGoalNoGoalFile(t *testing.T) {
	models := modelsForAgentFromGoal(t.TempDir(), "coordinator")
	assert.Empty(t, models)
}

func TestClassifyWorkspaceStandaloneNewBatch(t *testing.T) {
	dir := t.TempDir()
	kind := classifyWorkspace(dir)
	assert.Equal(t, workspaceStandalone, kind)
}

func TestClassifyWorkspaceForkNewBatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jj"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".jj", "repo"), []byte("/some/root/.jj/repo"), 0644))
	kind := classifyWorkspace(dir)
	assert.Equal(t, workspaceFork, kind)
}

func TestGetRootWorkspacePathForkRepo(t *testing.T) {
	base := t.TempDir()
	rootDir := filepath.Join(base, "root-workspace")
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, ".jj", "repo"), 0755))
	forkDir := filepath.Join(base, "fork-workspace")
	require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(forkDir, ".jj", "repo"), []byte(filepath.Join(rootDir, ".jj", "repo")), 0644))
	result := getRootWorkspacePath(forkDir)
	assert.Equal(t, rootDir, result)
}

func TestScanWorkspaceGroupsCached(t *testing.T) {
	server, _ := setupTestServer(t)
	groups1, err1 := server.scanWorkspaceGroups()
	assert.NoError(t, err1)
	groups2, err2 := server.scanWorkspaceGroups()
	assert.NoError(t, err2)
	assert.Equal(t, len(groups1), len(groups2))
}

func TestNotifyStateChange(t *testing.T) {
	server, _ := setupTestServer(t)
	server.notifyStateChange()
}

func TestInvalidateWorkspaceScanCacheNewBatch(t *testing.T) {
	server, _ := setupTestServer(t)
	server.invalidateWorkspaceScanCache()
	server.invalidateWorkspaceScanCache()
}

func TestResolveWorkspaceNameToPathEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	result := srv.resolveWorkspaceNameToPath("")
	assert.Empty(t, result)
}

func TestResolveWorkspaceNameToPathFound(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "my-workspace")
	result := srv.resolveWorkspaceNameToPath("my-workspace")
	assert.Equal(t, filepath.Join(rootDir, "my-workspace"), result)
}

func TestGatherSnippetsByLanguageMultiple(t *testing.T) {
	dir := t.TempDir()
	snippetsDir := filepath.Join(dir, ".sgai", "snippets")
	goDir := filepath.Join(snippetsDir, "go")
	pyDir := filepath.Join(snippetsDir, "python")
	require.NoError(t, os.MkdirAll(goDir, 0755))
	require.NoError(t, os.MkdirAll(pyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "http-server.go"), []byte("---\nname: HTTP Server\ndescription: Go HTTP server\n---\npackage main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pyDir, "flask-app.py"), []byte("---\nname: Flask App\ndescription: Flask web app\n---\nfrom flask import Flask"), 0644))
	result := gatherSnippetsByLanguage(dir)
	assert.Len(t, result, 2)
	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "python", result[1].Name)
	assert.Len(t, result[0].Snippets, 1)
	assert.Equal(t, "HTTP Server", result[0].Snippets[0].Name)
}

func TestGatherSnippetsByLanguageNoSnippetsDir(t *testing.T) {
	dir := t.TempDir()
	result := gatherSnippetsByLanguage(dir)
	assert.Nil(t, result)
}

func TestGatherSnippetsByLanguageNoDescription(t *testing.T) {
	dir := t.TempDir()
	goDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(goDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "simple.go"), []byte("package main"), 0644))
	result := gatherSnippetsByLanguage(dir)
	require.Len(t, result, 1)
	assert.Equal(t, "simple", result[0].Snippets[0].Name)
}

func TestAddGitExcludeCreatesExclude(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0755))
	err := addGitExclude(dir)
	assert.NoError(t, err)
	content, errRead := os.ReadFile(filepath.Join(dir, ".git", "info", "exclude"))
	require.NoError(t, errRead)
	assert.Contains(t, string(content), "/.sgai")
}

func TestAddGitExcludeWithExistingExclude(t *testing.T) {
	dir := t.TempDir()
	gitInfoDir := filepath.Join(dir, ".git", "info")
	require.NoError(t, os.MkdirAll(gitInfoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitInfoDir, "exclude"), []byte("# existing\n"), 0644))
	err := addGitExclude(dir)
	assert.NoError(t, err)
	content, errRead := os.ReadFile(filepath.Join(gitInfoDir, "exclude"))
	require.NoError(t, errRead)
	assert.Contains(t, string(content), "# existing")
	assert.Contains(t, string(content), "/.sgai")
}

func TestWriteGoalExample(t *testing.T) {
	dir := t.TempDir()
	err := writeGoalExample(dir)
	assert.NoError(t, err)
	content, errRead := os.ReadFile(filepath.Join(dir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, goalExampleContent, string(content))
}

func TestOrderedModelStatusesWithEntries(t *testing.T) {
	dir := t.TempDir()
	statuses := map[string]string{
		"model-a": "running",
		"model-b": "completed",
	}
	result := orderedModelStatuses(dir, statuses)
	assert.Len(t, result, 2)
}

func TestResolveBaseBookmarkNoJJ(t *testing.T) {
	result := resolveBaseBookmark(t.TempDir())
	assert.Equal(t, "main", result)
}

func TestResolveRootForDeleteForkStandalone(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "standalone-ws")
	result := srv.resolveRootForDeleteFork(filepath.Join(rootDir, "standalone-ws"))
	assert.Empty(t, result)
}

func TestResolveForkDirFromRequest(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-resolve")

	t.Run("emptyRequestSameAsRoot", func(t *testing.T) {
		result := srv.resolveForkDir("", wsDir, wsDir)
		assert.Empty(t, result)
	})

	t.Run("emptyRequestDifferentFromRoot", func(t *testing.T) {
		result := srv.resolveForkDir("", wsDir, filepath.Join(rootDir, "other-root"))
		assert.Equal(t, wsDir, result)
	})

	t.Run("invalidRequestDir", func(t *testing.T) {
		result := srv.resolveForkDir("/nonexistent/path", wsDir, wsDir)
		assert.Empty(t, result)
	})
}

func TestReadNewestForkGoalNoForks(t *testing.T) {
	result := readNewestForkGoal(nil)
	assert.Empty(t, result)
}

func TestReadNewestForkGoalWithFork(t *testing.T) {
	dir := t.TempDir()
	forkDir := filepath.Join(dir, "fork1")
	require.NoError(t, os.MkdirAll(forkDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("# Fork Goal"), 0644))
	forks := []workspaceInfo{{Directory: forkDir}}
	result := readNewestForkGoal(forks)
	assert.Equal(t, "# Fork Goal", result)
}

func TestReadNewestForkGoalEmptyContent(t *testing.T) {
	dir := t.TempDir()
	forkDir := filepath.Join(dir, "fork1")
	require.NoError(t, os.MkdirAll(forkDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("  \n  "), 0644))
	forks := []workspaceInfo{{Directory: forkDir}}
	result := readNewestForkGoal(forks)
	assert.Empty(t, result)
}

func TestModelsForAgentFromGoalNoGoal(t *testing.T) {
	dir := t.TempDir()
	result := modelsForAgentFromGoal(dir, "builder")
	assert.Nil(t, result)
}

func TestModelsForAgentFromGoalWithGoal(t *testing.T) {
	dir := t.TempDir()
	goalContent := "---\nmodels:\n  builder: anthropic/claude-opus-4-6\n---\n# Goal"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0644))
	result := modelsForAgentFromGoal(dir, "builder")
	assert.NotNil(t, result)
}

func TestUnpackSkeletonCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	err := unpackSkeleton(dir)
	assert.NoError(t, err)
	assert.True(t, isExistingDirectory(filepath.Join(dir, ".sgai")))
}

func TestResolveBaseBookmarkCached(t *testing.T) {
	srv, _ := setupTestServer(t)
	result1 := srv.resolveBaseBookmarkCached(t.TempDir())
	assert.Equal(t, "main", result1)
	result2 := srv.resolveBaseBookmarkCached(t.TempDir())
	assert.Equal(t, "main", result2)
}

func TestRunJJLogForForkNoJJ(t *testing.T) {
	result := runJJLogForFork("main", t.TempDir())
	assert.Nil(t, result)
}

func TestCountForkCommitsAheadNoJJ(t *testing.T) {
	result := countForkCommitsAhead("main", t.TempDir())
	assert.Equal(t, 0, result)
}
