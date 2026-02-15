package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type slackBotConfig struct {
	botToken     string
	appToken     string
	allowedUsers map[string]bool
	rootDir      string
}

func parseSlackBotConfig(rootDir string) (slackBotConfig, error) {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		return slackBotConfig{}, fmt.Errorf("SLACK_BOT_TOKEN environment variable is required")
	}

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		return slackBotConfig{}, fmt.Errorf("SLACK_APP_TOKEN environment variable is required")
	}

	allowedUsersStr := os.Getenv("SLACK_ALLOWED_USERS")
	if allowedUsersStr == "" {
		return slackBotConfig{}, fmt.Errorf("SLACK_ALLOWED_USERS environment variable is required")
	}

	allowedUsers := make(map[string]bool)
	for uid := range strings.SplitSeq(allowedUsersStr, ",") {
		trimmed := strings.TrimSpace(uid)
		if trimmed != "" {
			allowedUsers[trimmed] = true
		}
	}

	return slackBotConfig{
		botToken:     botToken,
		appToken:     appToken,
		allowedUsers: allowedUsers,
		rootDir:      rootDir,
	}, nil
}

func cmdSlackBot(args []string) {
	flagSet := flag.NewFlagSet("slack-bot", flag.ExitOnError)
	rootDir := flagSet.String("root-dir", "", "root directory for workspaces (defaults to CWD)")
	listenAddr := flagSet.String("listen-addr", "127.0.0.1:8080", "HTTP server listen address")
	flagSet.Parse(args) //nolint:errcheck

	dir := *rootDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			log.Fatalln("failed to get working directory:", err)
		}
	}

	cfg, err := parseSlackBotConfig(dir)
	if err != nil {
		log.Fatalln(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := NewServer(dir)
	srv.shutdownCtx = ctx
	if errLoad := srv.loadPinnedProjects(); errLoad != nil {
		log.Println("warning: failed to load pinned projects:", errLoad)
	}
	srv.startStateWatcher()

	go startInternalHTTPServer(ctx, srv, *listenAddr)

	bot := newSlackBot(cfg, srv)
	bot.run(ctx)
}

func startInternalHTTPServer(ctx context.Context, srv *Server, listenAddr string) {
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	handler := srv.spaMiddleware(mux)

	httpServer := &http.Server{Addr: listenAddr, Handler: handler}
	go func() {
		<-ctx.Done()
		if errClose := httpServer.Close(); errClose != nil {
			log.Println("http server close:", errClose)
		}
	}()
	log.Println("slack-bot internal HTTP server listening on", listenAddr)
	if errListen := httpServer.ListenAndServe(); errListen != nil && !errors.Is(errListen, http.ErrServerClosed) {
		log.Println("internal HTTP server error:", errListen)
	}
}

type slackBot struct {
	config     slackBotConfig
	server     *Server
	sessions   *sessionDB
	locks      *sessionLockMap
	api        *slack.Client
	socketMode *socketmode.Client
	replyCh    chan slackReplyMessage
}

func newSlackBot(cfg slackBotConfig, srv *Server) *slackBot {
	api := slack.New(cfg.botToken,
		slack.OptionAppLevelToken(cfg.appToken),
	)
	socketClient := socketmode.New(api)

	sessDB := newSessionDB(defaultSessionDBPath())
	if err := sessDB.load(); err != nil {
		log.Println("warning: loading session database:", err)
	}

	return &slackBot{
		config:     cfg,
		server:     srv,
		sessions:   sessDB,
		locks:      newSessionLockMap(),
		api:        api,
		socketMode: socketClient,
		replyCh:    make(chan slackReplyMessage, 64),
	}
}

func (b *slackBot) run(ctx context.Context) {
	watcher := newSlackNotificationWatcher(b.sessions, b.api)
	go watcher.run(ctx)

	go b.replyWorker(ctx)

	go func() {
		if err := b.socketMode.RunContext(ctx); err != nil {
			log.Println("slack socket mode error:", err)
		}
	}()

	log.Println("slack-bot connected and listening for events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("slack-bot shutting down...")
			return
		case evt := <-b.socketMode.Events:
			b.handleEvent(ctx, evt)
		}
	}
}

func (b *slackBot) replyWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-b.replyCh:
			b.postReply(msg)
		}
	}
}

func (b *slackBot) postReply(msg slackReplyMessage) {
	var opts []slack.MsgOption

	if msg.blocks != "" {
		var blocks slack.Blocks
		if errJSON := json.Unmarshal([]byte(msg.blocks), &blocks); errJSON != nil {
			opts = append(opts, slack.MsgOptionText(msg.text, false))
		} else {
			opts = append(opts, slack.MsgOptionBlocks(blocks.BlockSet...))
			if msg.text != "" {
				opts = append(opts, slack.MsgOptionText(msg.text, false))
			}
		}
	} else {
		text := msg.text
		if len(text) > 3000 {
			chunks := splitMessage(text, 3000)
			for i, chunk := range chunks {
				chunkOpts := []slack.MsgOption{
					slack.MsgOptionText(chunk, false),
					slack.MsgOptionTS(msg.threadTS),
				}
				if i > 0 {
					time.Sleep(200 * time.Millisecond)
				}
				if _, _, errPost := b.api.PostMessage(msg.channelID, chunkOpts...); errPost != nil {
					log.Println("posting slack chunk:", errPost)
				}
			}
			return
		}
		opts = append(opts, slack.MsgOptionText(text, false))
	}

	opts = append(opts, slack.MsgOptionTS(msg.threadTS))

	if _, _, errPost := b.api.PostMessage(msg.channelID, opts...); errPost != nil {
		log.Println("posting slack message:", errPost)
	}
}

func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	remaining := text
	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			chunks = append(chunks, remaining)
			break
		}
		splitIdx := strings.LastIndex(remaining[:maxLen], "\n")
		if splitIdx <= 0 {
			splitIdx = maxLen
		}
		chunks = append(chunks, remaining[:splitIdx])
		remaining = remaining[splitIdx:]
		if len(remaining) > 0 && remaining[0] == '\n' {
			remaining = remaining[1:]
		}
	}
	return chunks
}

func (b *slackBot) handleEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		b.socketMode.Ack(*evt.Request)
		b.handleEventsAPI(ctx, evt)
	case socketmode.EventTypeConnecting:
		log.Println("slack-bot: connecting...")
	case socketmode.EventTypeConnected:
		log.Println("slack-bot: connected")
	case socketmode.EventTypeConnectionError:
		log.Println("slack-bot: connection error")
	}
}

func (b *slackBot) handleEventsAPI(ctx context.Context, evt socketmode.Event) {
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		b.handleCallbackEvent(ctx, eventsAPIEvent)
	}
}

func (b *slackBot) handleCallbackEvent(ctx context.Context, evt slackevents.EventsAPIEvent) {
	switch innerEvt := evt.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		b.handleAppMention(ctx, innerEvt)
	case *slackevents.MessageEvent:
		b.handleMessage(ctx, innerEvt)
	}
}

func (b *slackBot) handleAppMention(ctx context.Context, evt *slackevents.AppMentionEvent) {
	if !b.isAllowed(evt.User) {
		return
	}

	threadTS := evt.ThreadTimeStamp
	if threadTS == "" {
		threadTS = evt.TimeStamp
	}

	text := stripBotMention(evt.Text)
	go b.routeMessage(ctx, evt.Channel, threadTS, text)
}

func (b *slackBot) handleMessage(ctx context.Context, evt *slackevents.MessageEvent) {
	if evt.SubType != "" {
		return
	}
	if !b.isAllowed(evt.User) {
		return
	}
	if evt.ThreadTimeStamp == "" {
		return
	}

	_, connected := b.sessions.get(evt.Channel, evt.ThreadTimeStamp)
	if !connected {
		return
	}

	go b.routeMessage(ctx, evt.Channel, evt.ThreadTimeStamp, evt.Text)
}

func (b *slackBot) isAllowed(userID string) bool {
	return b.config.allowedUsers[userID]
}

func (b *slackBot) routeMessage(ctx context.Context, channelID, threadTS, text string) {
	sess, connected := b.sessions.get(channelID, threadTS)

	var sessionID string
	if connected && sess.SessionID != "" {
		sessionID = sess.SessionID
	}

	lockKey := sessionKey(channelID, threadTS)
	b.locks.acquire(lockKey)
	defer b.locks.release(lockKey)

	mcpCtx := &slackBotMCPContext{
		rootDir:   b.config.rootDir,
		channelID: channelID,
		threadTS:  threadTS,
		replyCh:   b.replyCh,
		sessions:  b.sessions,
		server:    b.server,
	}

	mcpURL, mcpClose, errMCP := startSlackBotMCPServer(mcpCtx)
	if errMCP != nil {
		log.Println("starting slack-bot MCP server:", errMCP)
		b.replyCh <- slackReplyMessage{
			channelID: channelID,
			threadTS:  threadTS,
			text:      "Internal error: could not start MCP server.",
		}
		return
	}
	defer mcpClose()

	newSessionID, errRun := b.runOpenCodeAgent(ctx, text, sessionID, mcpURL)
	if errRun != nil {
		log.Println("running opencode agent:", errRun)
		b.replyCh <- slackReplyMessage{
			channelID: channelID,
			threadTS:  threadTS,
			text:      "I encountered an error processing your message. Please try again.",
		}

		if connected {
			sess.SessionID = ""
			if errPut := b.sessions.put(channelID, threadTS, sess); errPut != nil {
				log.Println("clearing failed session:", errPut)
			}
		}
		return
	}

	if newSessionID != "" && connected {
		if errUpdate := b.sessions.updateSessionID(channelID, threadTS, newSessionID); errUpdate != nil {
			log.Println("updating session ID:", errUpdate)
		}
	}
}

func (b *slackBot) runOpenCodeAgent(ctx context.Context, message, sessionID, mcpURL string) (string, error) {
	args := []string{"run", "--format=json", "--agent", "slack-frontdesk"}
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	}
	args = append(args, "--title", "slack-frontdesk")

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = b.config.rootDir
	cmd.Env = append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+filepath.Join(b.config.rootDir, ".sgai"),
		"SGAI_MCP_URL="+mcpURL,
		"SGAI_AGENT_IDENTITY=slack-frontdesk",
		"SGAI_MCP_INTERACTIVE=auto",
	)
	cmd.Stdin = strings.NewReader(message)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if errRun := cmd.Run(); errRun != nil {
		return "", fmt.Errorf("opencode run failed: %w", errRun)
	}

	return extractSessionIDFromOutput(stdout.String()), nil
}

func extractSessionIDFromOutput(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event streamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.SessionID != "" {
			return event.SessionID
		}
	}
	return ""
}

func stripBotMention(text string) string {
	result := text
	for strings.Contains(result, "<@") {
		start := strings.Index(result, "<@")
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}
