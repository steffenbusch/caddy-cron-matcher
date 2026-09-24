// Copyright 2024 Steffen Busch

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cronmatcher

import (
	"net/http"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func setupLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	return logger, logs
}

func TestCronMatcher_Match(t *testing.T) {
	logger, _ := setupLogger()

	// Sample matcher setup
	cm := &CronMatcher{
		EnableAt:  []string{"0 10 * * 1-5", "0 15 * * 0,6"}, // Matches at 10:00 on weekdays and 15:00 on weekends
		DisableAt: []string{"0 11 * * 1-5", "0 16 * * 0,6"}, // Ends at 11:00 on weekdays and 16:00 on weekends
		logger:    logger,
	}

	// Provision step to check that the matcher sets up correctly
	if err := cm.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provisioning failed: %v", err)
	}

	r, _ := http.NewRequest("GET", "http://example.com", nil)

	// Set UTC location for consistent time testing
	location := time.UTC

	// Test case 1: Time within the first enable/disable range on a weekday
	mockTime, _ := time.ParseInLocation("2006-01-02 15:04", "2024-11-01 10:30", location) // A weekday
	nowFunc = func() time.Time { return mockTime }
	if !cm.Match(r) {
		t.Error("Expected request to match within the first enable/disable time window on a weekday (10:30)")
	}

	// Test case 2: Time outside all enable/disable windows
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-01 12:30", location) // Outside the defined ranges
	nowFunc = func() time.Time { return mockTime }
	if cm.Match(r) {
		t.Error("Expected request not to match outside the enable/disable time windows (12:30)")
	}

	// Test case 3: Time at the exact start of an enable time on a weekday
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-01 10:00", location) // Exactly at the start of the first enable window on a weekday
	nowFunc = func() time.Time { return mockTime }
	if !cm.Match(r) {
		t.Error("Expected request to match exactly at the start of the enable time on a weekday (10:00)")
	}

	// Test case 4: Time at the exact end of an enable window on a weekday
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-01 11:00", location) // Exactly at the end of the first enable window on a weekday
	nowFunc = func() time.Time { return mockTime }
	if cm.Match(r) {
		t.Error("Expected request not to match exactly at the end of the disable time on a weekday (11:00)")
	}

	// Test case 5: Time at the exact start of the second enable time on a weekend
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-03 15:00", location) // Exactly at the start of the second enable window on a weekend (Sunday)
	nowFunc = func() time.Time { return mockTime }
	if !cm.Match(r) {
		t.Error("Expected request to match exactly at the start of the second enable time on a weekend (15:00)")
	}

	// Test case 6: Time at the exact end of the second enable window on a weekend
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-03 16:00", location) // Exactly at the end of the second enable window on a weekend (Sunday)
	nowFunc = func() time.Time { return mockTime }
	if cm.Match(r) {
		t.Error("Expected request not to match exactly at the end of the disable time on a weekend (16:00)")
	}

	// Test case 7: Time within the enable/disable window on a weekend
	mockTime, _ = time.ParseInLocation("2006-01-02 15:04", "2024-11-03 15:30", location) // Time during a weekend window (Sunday)
	nowFunc = func() time.Time { return mockTime }
	if !cm.Match(r) {
		t.Error("Expected request to match during the enable window on a weekend (15:30)")
	}

	// Restore the original time function after the tests
	defer func() { nowFunc = time.Now }()
}

func TestCronMatcher_MatchAcrossBerlinDST(t *testing.T) {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin location: %v", err)
	}

	for _, test := range []struct {
		name        string
		enableAt    string
		disableAt   string
		now         time.Time
		expectMatch bool
	}{
		{
			// Maintenance starts on Friday, 27 March 2026, at 22:00 CET.
			// DST starts on Sunday, 29 March: 02:00 jumps to 03:00.
			// Maintenance ends on Monday, 30 March, at 05:00 CEST.
			// It must therefore be active on Sunday, 29 March, at noon.
			name:        "summer time: Sunday 29 March 2026 at 12:00 CEST is active",
			enableAt:    "0 22 * * 5",
			disableAt:   "0 5 * * 1",
			now:         time.Date(2026, time.March, 29, 12, 0, 0, 0, location),
			expectMatch: true,
		},
		{
			// The same window ends exactly at Monday, 30 March 2026, 05:00 CEST.
			// disableAt is an exclusive boundary, so the matcher must be inactive.
			name:        "summer time: Monday 30 March 2026 at 05:00 CEST is inactive",
			enableAt:    "0 22 * * 5",
			disableAt:   "0 5 * * 1",
			now:         time.Date(2026, time.March, 30, 5, 0, 0, 0, location),
			expectMatch: false,
		},
		{
			// Maintenance starts on Friday, 23 October 2026, at 22:00 CEST.
			// DST ends on Sunday, 25 October: 03:00 returns to 02:00.
			// Maintenance ends on Monday, 26 October, at 05:00 CET.
			// It must therefore be active on Sunday, 25 October, at noon.
			name:        "winter time: Sunday 25 October 2026 at 12:00 CET is active",
			enableAt:    "0 22 * * 5",
			disableAt:   "0 5 * * 1",
			now:         time.Date(2026, time.October, 25, 12, 0, 0, 0, location),
			expectMatch: true,
		},
		{
			// The same window ends exactly at Monday, 26 October 2026, 05:00 CET.
			// disableAt is an exclusive boundary, so the matcher must be inactive.
			name:        "winter time: Monday 26 October 2026 at 05:00 CET is inactive",
			enableAt:    "0 22 * * 5",
			disableAt:   "0 5 * * 1",
			now:         time.Date(2026, time.October, 26, 5, 0, 0, 0, location),
			expectMatch: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// enableAt is every Friday at 22:00; disableAt is every Monday at 05:00.
			cm := &CronMatcher{
				EnableAt:  []string{test.enableAt},
				DisableAt: []string{test.disableAt},
				logger:    zap.NewNop(),
			}

			previousNowFunc := nowFunc
			nowFunc = func() time.Time { return test.now }
			t.Cleanup(func() { nowFunc = previousNowFunc })

			r, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			if got := cm.Match(r); got != test.expectMatch {
				t.Fatalf("Match() = %t, want %t", got, test.expectMatch)
			}
		})
	}
}

func TestCronMatcher_MatchAcrossBerlinDSTRegression(t *testing.T) {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin location: %v", err)
	}

	// The last enable tick is Saturday, 28 March 2026, 02:00 CET.
	// The next disable tick must be Saturday, 4 April 2026, 02:00 CEST.
	// gronx v1.20.3 incorrectly returned Sunday, 29 March, after the DST jump.
	cm := &CronMatcher{
		EnableAt:  []string{"0 2 * 3 6"}, // Every Saturday in March at 02:00.
		DisableAt: []string{"0 2 * * 6"}, // Every Saturday at 02:00.
		logger:    zap.NewNop(),
	}

	// Monday, 30 March 2026, 12:00 CEST is after the DST change and before
	// the expected disable tick on Saturday, 4 April.
	now := time.Date(2026, time.March, 30, 12, 0, 0, 0, location)
	previousNowFunc := nowFunc
	nowFunc = func() time.Time { return now }
	t.Cleanup(func() { nowFunc = previousNowFunc })

	r, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if !cm.Match(r) {
		t.Fatal("expected matcher to remain active until Saturday, 4 April 2026, 02:00 CEST")
	}
}
