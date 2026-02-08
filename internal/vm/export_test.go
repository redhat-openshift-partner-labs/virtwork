// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package vm

import "time"

// SetBaseRetryBackoff overrides the retry backoff duration for testing.
// Returns a function that restores the original value.
func SetBaseRetryBackoff(d time.Duration) func() {
	old := baseRetryBackoff
	baseRetryBackoff = d
	return func() { baseRetryBackoff = old }
}
