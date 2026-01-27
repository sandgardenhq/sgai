package notify

import "log"

// Send sends notifications to all available destinations.
// It attempts both local (platform-specific) and remote (ntfy) notifications.
// Failures are logged but do not affect program flow.
func Send(title, message string) {
	senders := []func(string, string) error{
		sendLocal,
		sendRemote,
	}
	for _, send := range senders {
		errSend := send(title, message)
		if errSend != nil {
			log.Println("notification failed:", errSend)
		}
	}
}
