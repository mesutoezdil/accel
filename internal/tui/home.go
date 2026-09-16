package tui

import "os"

func userHome() (string, error) { return os.UserHomeDir() }
