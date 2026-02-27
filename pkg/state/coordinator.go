package state

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const agentDoneWatchdogTimeout = time.Minute

// Coordinator manages workflow state in memory with blocking ask/answer delivery
// and a soft-stop watchdog for agent-done transitions.
//
// It wraps a Workflow with channels for direct answer delivery (no file I/O),
// a save path for retrospective persistence, and a sync.Once-guarded timer
// that cancels the context one minute after agent-done is called.
type Coordinator struct {
	mu       sync.Mutex
	wf       Workflow
	answerCh chan string
	savePath string

	doneOnce  sync.Once
	doneTimer *time.Timer

	onUpdate func()
}

// NewCoordinator creates a Coordinator for the given state.json path.
// It loads the initial workflow state from disk exactly once; all subsequent
// state access is in-memory via the returned Coordinator.
func NewCoordinator(path string) (*Coordinator, error) {
	wf, err := load(path)
	if err != nil {
		return nil, fmt.Errorf("loading coordinator state: %w", err)
	}
	return &Coordinator{
		wf:       wf,
		answerCh: make(chan string, 1),
		savePath: path,
	}, nil
}

// NewCoordinatorEmpty creates a Coordinator with an empty workflow state for the given path.
// Use this when state.json does not yet exist and the workflow is starting fresh.
func NewCoordinatorEmpty(path string) *Coordinator {
	return &Coordinator{
		wf: Workflow{
			Status:   StatusWorking,
			Progress: []ProgressEntry{},
			Messages: []Message{},
		},
		answerCh: make(chan string, 1),
		savePath: path,
	}
}

// NewCoordinatorWith creates a Coordinator seeded with the given workflow state and
// persists it to disk immediately. Use this in tests and setup code that needs to
// establish a known on-disk state before a Coordinator is loaded by the server.
func NewCoordinatorWith(path string, wf Workflow) (*Coordinator, error) {
	c := &Coordinator{
		wf:       wf,
		answerCh: make(chan string, 1),
		savePath: path,
	}
	if err := save(path, wf); err != nil {
		return nil, fmt.Errorf("saving initial coordinator state: %w", err)
	}
	return c, nil
}

// State returns a snapshot of the current workflow under the coordinator lock.
func (c *Coordinator) State() Workflow {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wf
}

// OnUpdate registers a callback that fires after every successful UpdateState.
// The callback is called outside the coordinator lock and is safe to use for
// invalidating UI caches or publishing SSE events.
func (c *Coordinator) OnUpdate(fn func()) {
	c.mu.Lock()
	c.onUpdate = fn
	c.mu.Unlock()
}

// UpdateState applies fn to the workflow under the coordinator lock and saves
// the result to disk for retrospective persistence.
func (c *Coordinator) UpdateState(fn func(*Workflow)) error {
	c.mu.Lock()
	fn(&c.wf)
	snapshot := c.wf
	notify := c.onUpdate
	c.mu.Unlock()
	if err := save(c.savePath, snapshot); err != nil {
		return err
	}
	if notify != nil {
		notify()
	}
	return nil
}

// AskAndWait sets the question state on the workflow, saves it, and blocks until
// the human partner answers via Respond or the context is cancelled.
// After the answer arrives, it clears the waiting state before returning.
// It returns the human's answer string or a context error.
func (c *Coordinator) AskAndWait(ctx context.Context, question *MultiChoiceQuestion, humanMessage string) (string, error) {
	if err := c.UpdateState(func(wf *Workflow) {
		wf.MultiChoiceQuestion = question
		wf.HumanMessage = humanMessage
		wf.Status = StatusWaitingForHuman
	}); err != nil {
		return "", fmt.Errorf("saving question state: %w", err)
	}

	clearWaiting := func() {
		if err := c.UpdateState(func(wf *Workflow) {
			wf.MultiChoiceQuestion = nil
			wf.HumanMessage = ""
			if IsHumanPending(wf.Status) {
				wf.Status = StatusWorking
			}
		}); err != nil {
			log.Println("failed to clear waiting state:", err)
		}
	}

	var answer string
	select {
	case <-ctx.Done():
		clearWaiting()
		return "", ctx.Err()
	case answer = <-c.answerCh:
	}

	clearWaiting()
	return answer, nil
}

// Respond delivers the human's answer to the blocked AskAndWait call.
// The state is cleared by AskAndWait after it receives the answer.
// Respond does not block.
func (c *Coordinator) Respond(answer string) {
	select {
	case c.answerCh <- answer:
	default:
	}
}

// StartAgentDoneWatchdog starts a one-minute timer that calls cancel once.
// Repeated calls are silently ignored via sync.Once.
func (c *Coordinator) StartAgentDoneWatchdog(cancel context.CancelFunc) {
	c.doneOnce.Do(func() {
		c.mu.Lock()
		c.doneTimer = time.AfterFunc(agentDoneWatchdogTimeout, cancel)
		c.mu.Unlock()
	})
}

// IsShuttingDown reports whether the agent-done watchdog has been started.
func (c *Coordinator) IsShuttingDown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.doneTimer != nil
}

// Stop cancels the watchdog timer if it is running.
func (c *Coordinator) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.doneTimer != nil {
		c.doneTimer.Stop()
	}
}
