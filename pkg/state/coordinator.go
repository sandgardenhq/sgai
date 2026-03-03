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
// It wraps a Workflow with per-question response channels for direct answer
// delivery (no file I/O), a save path for retrospective persistence, and a
// sync.Once-guarded timer that cancels the context one minute after agent-done
// is called.
type Coordinator struct {
	mu                sync.Mutex
	wf                Workflow
	currentResponseCh chan string
	savePath          string

	doneOnce    sync.Once
	doneTimer   *time.Timer
	agentCancel context.CancelFunc

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
		savePath: path,
	}
}

// NewCoordinatorWith creates a Coordinator seeded with the given workflow state and
// persists it to disk immediately. Use this in tests and setup code that needs to
// establish a known on-disk state before a Coordinator is loaded by the server.
func NewCoordinatorWith(path string, wf Workflow) (*Coordinator, error) {
	c := &Coordinator{
		wf:       wf,
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
// On context cancellation, the question state is preserved so the UI
// notification (⚠) remains visible for the human partner, and the response
// channel is kept alive so a subsequent call can immediately collect a buffered
// answer.
// It returns the human's answer string or a context error.
//
// Each call creates its own response channel (channel-in-channel pattern) so
// that concurrent or sequential calls never share state and MCP tool timeouts
// do not corrupt pending question channels. If the previous call timed out and
// the human has already answered (buffered in the existing channel), the answer
// is collected immediately without blocking.
func (c *Coordinator) AskAndWait(ctx context.Context, question *MultiChoiceQuestion, humanMessage string) (string, error) {
	c.mu.Lock()
	existingCh := c.currentResponseCh
	c.mu.Unlock()

	if existingCh != nil {
		select {
		case buffered := <-existingCh:
			log.Println("askandwait: collected buffered answer from previous call")
			c.mu.Lock()
			if c.currentResponseCh == existingCh {
				c.currentResponseCh = nil
			}
			c.mu.Unlock()
			c.clearWaitingState()
			return buffered, nil
		default:
		}
	}

	responseCh := make(chan string, 1)

	c.mu.Lock()
	c.currentResponseCh = responseCh
	c.mu.Unlock()

	if err := c.UpdateState(func(wf *Workflow) {
		wf.MultiChoiceQuestion = question
		wf.HumanMessage = humanMessage
		wf.Status = StatusWaitingForHuman
	}); err != nil {
		c.mu.Lock()
		if c.currentResponseCh == responseCh {
			c.currentResponseCh = nil
		}
		c.mu.Unlock()
		return "", fmt.Errorf("saving question state: %w", err)
	}
	log.Println("askandwait: question state set, status changed to waiting-for-human")

	log.Println("askandwait: blocking for human answer")
	var answer string
	select {
	case <-ctx.Done():
		log.Println("askandwait: context cancelled:", ctx.Err())
		return "", ctx.Err()
	case answer = <-responseCh:
		log.Println("askandwait: answer received from human")
	}

	c.mu.Lock()
	if c.currentResponseCh == responseCh {
		c.currentResponseCh = nil
	}
	c.mu.Unlock()

	c.clearWaitingState()
	return answer, nil
}

func (c *Coordinator) clearWaitingState() {
	log.Println("askandwait: clearing waiting state")
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

// Respond delivers the human's answer to the blocked AskAndWait call.
// The state is cleared by AskAndWait after it receives the answer.
// Respond does not block. If no AskAndWait is currently waiting, the answer
// is silently discarded.
func (c *Coordinator) Respond(answer string) {
	c.mu.Lock()
	responseCh := c.currentResponseCh
	c.mu.Unlock()

	if responseCh == nil {
		log.Println("askandwait: no pending question, discarding response")
		return
	}

	select {
	case responseCh <- answer:
		log.Println("askandwait: response queued for delivery")
	default:
		log.Println("askandwait: response channel full, response discarded")
	}
}

// SetAgentCancel stores the cancel function for the current agent run.
// It is called before each agent subprocess is launched so the watchdog can
// terminate that specific run if it hangs after setting status:agent-done.
func (c *Coordinator) SetAgentCancel(cancel context.CancelFunc) {
	c.mu.Lock()
	c.agentCancel = cancel
	c.mu.Unlock()
}

// GetAgentCancel returns the cancel function for the current agent run.
// Returns nil if none has been set or after ResetAgentDoneWatchdog clears it.
func (c *Coordinator) GetAgentCancel() context.CancelFunc {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.agentCancel
}

// ResetAgentDoneWatchdog prepares the watchdog for a fresh agent run.
// It stops any pending timer, clears the stored cancel function, and resets
// the sync.Once so the watchdog can fire again on the next agent-done.
func (c *Coordinator) ResetAgentDoneWatchdog() {
	c.mu.Lock()
	if c.doneTimer != nil {
		c.doneTimer.Stop()
		c.doneTimer = nil
	}
	c.agentCancel = nil
	c.doneOnce = sync.Once{}
	c.mu.Unlock()
}

// StartAgentDoneWatchdog starts a one-minute timer that calls cancel once.
// Repeated calls are silently ignored via sync.Once.
func (c *Coordinator) StartAgentDoneWatchdog(cancel context.CancelFunc) {
	if cancel == nil {
		return
	}
	c.mu.Lock()
	c.doneOnce.Do(func() {
		c.doneTimer = time.AfterFunc(agentDoneWatchdogTimeout, cancel)
	})
	c.mu.Unlock()
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
