//go:build !darwin

package main

import "fmt"

func openInTerminal(_ string) error {
	return fmt.Errorf("open in terminal not supported on this platform")
}
