package state

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	logFunc  func(string)
}

// NewCoordinator creates a Coordinator for the given state.json path.
// It loads the initial workflow state from disk exactly once; all subsequent
// state access is in-memory via the returned Coordinator.
func NewCoordinator(path string) (*Coordinator, error) {
	wf, err := load(path)
	if err != nil {
		return nil, fmt.Errorf("loading coordinator state: %w", err)
	}
	wf = transientFreeWorkflow(wf)
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
	if err := save(path, transientFreeWorkflow(wf)); err != nil {
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

// SetLogFunc registers a callback for coordinator diagnostic log lines.
func (c *Coordinator) SetLogFunc(fn func(string)) {
	c.mu.Lock()
	c.logFunc = fn
	c.mu.Unlock()
}

func (c *Coordinator) log(args ...any) {
	c.mu.Lock()
	logFunc := c.logFunc
	c.mu.Unlock()

	if logFunc != nil {
		logFunc(strings.TrimSpace(fmt.Sprintln(args...)))
		return
	}
	log.Println(args...)
}

// UpdateState applies fn to the workflow under the coordinator lock and saves
// the result to disk for retrospective persistence.
func (c *Coordinator) UpdateState(fn func(*Workflow)) error {
	c.mu.Lock()
	fn(&c.wf)
	snapshot := c.wf
	persistent := transientFreeWorkflow(snapshot)
	notify := c.onUpdate
	c.mu.Unlock()
	if err := save(c.savePath, persistent); err != nil {
		return err
	}
	if notify != nil {
		notify()
	}
	return nil
}

func (c *Coordinator) updateTransientState(fn func(*Workflow)) {
	c.mu.Lock()
	fn(&c.wf)
	notify := c.onUpdate
	c.mu.Unlock()
	if notify != nil {
		notify()
	}
}

func transientFreeWorkflow(wf Workflow) Workflow {
	wf.HumanMessage = ""
	wf.MultiChoiceQuestion = nil
	if wf.Status == StatusWaitingForHuman {
		wf.Status = StatusWorking
	}
	return wf
}

// AskAndWait sets the question state in memory and blocks until the human
// partner answers via Respond or the context is cancelled.
// After the answer arrives or the context is cancelled, it clears the waiting
// state before returning.
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
			c.log("askandwait: collected buffered answer from previous call")
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

	c.updateTransientState(func(wf *Workflow) {
		wf.MultiChoiceQuestion = question
		wf.HumanMessage = humanMessage
		wf.Status = StatusWaitingForHuman
	})
	c.log("askandwait: question state set, status changed to waiting-for-human")

	c.log("askandwait: blocking for human answer")
	var answer string
	select {
	case <-ctx.Done():
		c.log("askandwait: context cancelled:", ctx.Err())
		c.mu.Lock()
		if c.currentResponseCh == responseCh {
			c.currentResponseCh = nil
		}
		c.mu.Unlock()
		c.clearWaitingState()
		return "", ctx.Err()
	case answer = <-responseCh:
		c.log("askandwait: answer received from human")
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
	c.log("askandwait: clearing waiting state")
	c.updateTransientState(func(wf *Workflow) {
		wf.MultiChoiceQuestion = nil
		wf.HumanMessage = ""
		if IsHumanPending(wf.Status) {
			wf.Status = StatusWorking
		}
	})
}

// Respond delivers the human's answer to the blocked AskAndWait call.
// The state is cleared by AskAndWait after it receives the answer.
// Respond does not block. It reports whether the answer was queued for delivery.
func (c *Coordinator) Respond(answer string) bool {
	c.mu.Lock()
	responseCh := c.currentResponseCh
	c.mu.Unlock()

	if responseCh == nil {
		c.log("askandwait: no pending question, discarding response")
		return false
	}

	select {
	case responseCh <- answer:
		c.log("askandwait: response queued for delivery")
		return true
	default:
		c.log("askandwait: response channel full, response discarded")
		return false
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

// Stop cancels the watchdog timer if it is running and clears transient state.
func (c *Coordinator) Stop() {
	c.mu.Lock()
	if c.doneTimer != nil {
		c.doneTimer.Stop()
	}
	c.currentResponseCh = nil
	c.wf.MultiChoiceQuestion = nil
	c.wf.HumanMessage = ""
	if IsHumanPending(c.wf.Status) {
		c.wf.Status = StatusWorking
	}
	notify := c.onUpdate
	c.mu.Unlock()
	if notify != nil {
		notify()
	}
}
