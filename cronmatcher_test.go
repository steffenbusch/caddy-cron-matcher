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
