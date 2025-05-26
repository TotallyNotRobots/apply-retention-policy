/*
The MIT License (MIT)

Copyright © 2025 linuxdaemon <linuxdaemon.irc@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package util provides various utilities for the application.
package util

import (
	"errors"
)

// ErrNotImplemented is returned when a platform-specific operation is not implemented
var ErrNotImplemented = errors.New("operation not implemented on this platform")

// FileSystemStats contains filesystem statistics
type FileSystemStats struct {
	Type int64
}

// Platform provides platform-specific file operations
type Platform interface {
	// Statfs retrieves filesystem statistics
	Statfs(path string, stat *FileSystemStats) error
	// Mkfifo creates a named pipe (FIFO)
	Mkfifo(path string, mode uint32) error
	// GetSupportedFsTypes returns a list of supported filesystem types
	GetSupportedFsTypes() []int64
	// CheckACLSupport checks if the system supports ACLs
	CheckACLSupport() (bool, error)
	// CheckSymlinkSupport checks if the system supports symlinks
	CheckSymlinkSupport() (bool, error)
	// CheckFIFOSupport checks if the system supports named pipes (FIFOs)
	CheckFIFOSupport() (bool, error)
	// SetReadOnly makes a file read-only
	SetReadOnly(path string) error
	// RemoveReadOnly removes read-only attribute from a file
	RemoveReadOnly(path string) error
}

// Platform-specific implementations are in separate files with build tags
