package notification

import "fmt"

// StdoutNotifier is a simple notifier that prints to stdout (for testing)
type StdoutNotifier struct{}

// NewStdoutNotifier creates a new stdout notifier
func NewStdoutNotifier() *StdoutNotifier {
	return &StdoutNotifier{}
}

// Send prints the notification to stdout
func (n *StdoutNotifier) Send(notification Notification) error {
	fmt.Printf("[NOTIFICATION] %s: %s (Pattern: %s)\n",
		notification.Title,
		notification.Message,
		notification.Pattern)
	return nil
}
