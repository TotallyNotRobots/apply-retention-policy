//go:build linux
// +build linux

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

package files

import (
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// LinuxPlatform implements Platform for Linux systems
type LinuxPlatform struct{}

// NewPlatform returns a new Platform implementation for the current system
func NewPlatform() Platform {
	return &LinuxPlatform{}
}

// Statfs implements Platform.Statfs for Linux systems
func (p *LinuxPlatform) Statfs(path string, stat *FileSystemStats) error {
	var unixStat syscall.Statfs_t

	err := syscall.Statfs(path, &unixStat)
	if err != nil {
		return err
	}

	stat.Type = unixStat.Type

	return nil
}

// Mkfifo implements Platform.Mkfifo for Linux systems
func (p *LinuxPlatform) Mkfifo(path string, mode uint32) error {
	return syscall.Mkfifo(path, mode)
}

// GetSupportedFsTypes implements Platform.GetSupportedFsTypes for Linux systems
func (p *LinuxPlatform) GetSupportedFsTypes() []int64 {
	return []int64{
		int64(unix.XFS_SUPER_MAGIC),
		int64(unix.EXT4_SUPER_MAGIC),
		int64(unix.BTRFS_SUPER_MAGIC),
	}
}

// CheckACLSupport implements Platform.CheckACLSupport for Linux systems
func (p *LinuxPlatform) CheckACLSupport() (bool, error) {
	var stat FileSystemStats

	err := p.Statfs(".", &stat)
	if err != nil {
		return false, err
	}

	for _, fsType := range p.GetSupportedFsTypes() {
		if stat.Type == fsType {
			return true, nil
		}
	}

	return false, nil
}

// CheckSymlinkSupport implements Platform.CheckSymlinkSupport for Linux systems
func (p *LinuxPlatform) CheckSymlinkSupport() (bool, error) {
	return true, nil
}

// CheckFIFOSupport implements Platform.CheckFIFOSupport for Linux systems
func (p *LinuxPlatform) CheckFIFOSupport() (bool, error) {
	return true, nil
}

// SetReadOnly implements Platform.SetReadOnly for Linux systems
func (p *LinuxPlatform) SetReadOnly(path string) error {
	return os.Chmod(filepath.Clean(path), 0o400)
}

// RemoveReadOnly implements Platform.RemoveReadOnly for Linux systems
func (p *LinuxPlatform) RemoveReadOnly(path string) error {
	return os.Chmod(filepath.Clean(path), 0o600)
}
