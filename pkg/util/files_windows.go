//go:build windows
// +build windows

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

package util

import (
	"os/exec"
	"path/filepath"
)

// WindowsPlatform implements Platform for Windows systems
type WindowsPlatform struct{}

// NewPlatform returns a new Platform implementation for the current system
func NewPlatform() Platform {
	return &WindowsPlatform{}
}

// Statfs implements Platform.Statfs for Windows
func (p *WindowsPlatform) Statfs(path string, stat *FileSystemStats) error {
	return ErrNotImplemented
}

// Mkfifo implements Platform.Mkfifo for Windows
func (p *WindowsPlatform) Mkfifo(path string, mode uint32) error {
	return ErrNotImplemented
}

// GetSupportedFsTypes implements Platform.GetSupportedFsTypes for Windows
func (p *WindowsPlatform) GetSupportedFsTypes() []int64 {
	return nil
}

// CheckACLSupport implements Platform.CheckACLSupport for Windows
func (p *WindowsPlatform) CheckACLSupport() (bool, error) {
	return false, nil
}

// CheckSymlinkSupport implements Platform.CheckSymlinkSupport for Windows
func (p *WindowsPlatform) CheckSymlinkSupport() (bool, error) {
	return false, nil
}

// CheckFIFOSupport implements Platform.CheckFIFOSupport for Windows
func (p *WindowsPlatform) CheckFIFOSupport() (bool, error) {
	return false, nil
}

// SetReadOnly implements Platform.SetReadOnly for Windows
func (p *WindowsPlatform) SetReadOnly(path string) error {
	// #nosec G204 - path is cleaned with filepath.Clean before use
	cmd := exec.Command("attrib", "+R", filepath.Clean(path))
	return cmd.Run()
}

// RemoveReadOnly implements Platform.RemoveReadOnly for Windows
func (p *WindowsPlatform) RemoveReadOnly(path string) error {
	// #nosec G204 - path is cleaned with filepath.Clean before use
	cmd := exec.Command("attrib", "-R", filepath.Clean(path))
	return cmd.Run()
}
