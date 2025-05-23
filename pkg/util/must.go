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

// Package util provides utility functions for common operations.
// It includes functions for handling errors and other common tasks.
package util

// Must panics if the given error is not nil.
// This is useful for handling errors that should
// never occur in normal operation.
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustReturn returns the value if err is nil, otherwise it panics.
// This is useful for handling errors that should never occur in normal
// operation, while also returning a value in the success case.
func MustReturn[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}

	return val
}
