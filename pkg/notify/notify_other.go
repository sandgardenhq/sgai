//go:build !darwin

package notify

func sendLocal(title, message string) error {
	return nil
}
