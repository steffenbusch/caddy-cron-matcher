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
	"fmt"
	"net/http"
	"time"

	"github.com/adhocore/gronx"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

// nowFunc allows us to override time.Now for testing purposes
var nowFunc = time.Now

func init() {
	caddy.RegisterModule(CronMatcher{})
}

// CronMatcher matches requests based on multiple sets of cron expressions.
// It allows you to define multiple time windows during which requests should be matched.
// The matcher becomes active after any of the time windows specified by EnableAt
// and inactive after any corresponding DisableAt.
type CronMatcher struct {
	// EnableAt is a slice of cron expressions specifying when the matcher should start matching.
	// Each entry in the slice corresponds to a matching time window.
	EnableAt []string `json:"enable_at,omitempty"`

	// DisableAt is a slice of cron expressions specifying when the matcher should stop matching.
	// Each entry in the slice must correspond to an entry in EnableAt.
	DisableAt []string `json:"disable_at,omitempty"`

	// logger is used for logging within the module.
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (CronMatcher) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.matchers.cron",
		New: func() caddy.Module { return new(CronMatcher) },
	}
}

// Provision sets up the CronMatcher.
// It ensures that EnableAt and DisableAt contain the same number of entries
// and logs each configured cron schedule.
func (cm *CronMatcher) Provision(ctx caddy.Context) error {
	cm.logger = ctx.Logger()

	// Ensure both EnableAt and DisableAt are defined and contain entries.
	if len(cm.EnableAt) == 0 || len(cm.DisableAt) == 0 {
		return fmt.Errorf("both 'enable_at' and 'disable_at' must contain at least one cron expression")
	}

	// Ensure the number of EnableAt and DisableAt entries match.
	if len(cm.EnableAt) != len(cm.DisableAt) {
		return fmt.Errorf("'enable_at' and 'disable_at' must have the same number of cron expressions")
	}

	// Log the configured cron expressions for each set.
	for i := 0; i < len(cm.EnableAt); i++ {
		cm.logger.Info(fmt.Sprintf("CronMatcher configured (set %d, OR'ed together)", i+1),
			zap.String("EnableAt", cm.EnableAt[i]),
			zap.String("DisableAt", cm.DisableAt[i]),
		)
	}

	return nil
}

// Validate checks all provided cron expressions in EnableAt and DisableAt.
// It ensures that each cron expression is valid.
func (cm *CronMatcher) Validate() error {
	// Validate each EnableAt cron expression.
	for i, enableAt := range cm.EnableAt {
		if !gronx.IsValid(enableAt) {
			return fmt.Errorf("invalid enable_at cron format at index %d: '%s'", i, enableAt)
		}
	}

	// Validate each DisableAt cron expression.
	for i, disableAt := range cm.DisableAt {
		if !gronx.IsValid(disableAt) {
			return fmt.Errorf("invalid disable_at cron format at index %d: '%s'", i, disableAt)
		}
	}

	return nil
}

// UnmarshalCaddyfile sets up the module from Caddyfile tokens.
// This method handles multiple `cron` blocks, appending each pair of arguments
// as a new entry in the EnableAt and DisableAt slices.
func (cm *CronMatcher) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() { // loop to handle multiple cron directives
		args := d.RemainingArgs()
		if len(args) == 2 {
			cm.EnableAt = append(cm.EnableAt, args[0])
			cm.DisableAt = append(cm.DisableAt, args[1])
		} else {
			return d.Err("each 'cron' block must have exactly 2 arguments: 'enable_at' and 'disable_at'")
		}
	}
	return nil
}

// Match determines whether the current request should be matched based on any of the cron schedules.
// It checks if the current time is between the last time an EnableAt cron expression matched
// and the next time the corresponding DisableAt cron expression will match.
func (cm *CronMatcher) Match(r *http.Request) bool {
	now := nowFunc() // Use nowFunc to enable mocking during tests

	for i := range cm.EnableAt {
		// Set 'inclRefTime' to true to ensure that 'now' is included as a valid tick if it exactly aligns with the cron-specified time.
		// This allows 'now' to be considered the most recent tick when evaluating time windows.
		lastEnable, err := gronx.PrevTickBefore(cm.EnableAt[i], now, true)
		if err != nil {
			cm.logger.Error("Failed to compute last enable time", zap.String("EnableAt", cm.EnableAt[i]), zap.Error(err))
			continue
		}

		nextDisable, err := gronx.NextTickAfter(cm.DisableAt[i], lastEnable, false)
		if err != nil {
			cm.logger.Error("Failed to compute next disable time", zap.String("DisableAt", cm.DisableAt[i]), zap.Error(err))
			continue
		}

		cm.logger.Debug("Evaluating cron schedule",
			zap.String("EnableAt", cm.EnableAt[i]),
			zap.String("DisableAt", cm.DisableAt[i]),
			zap.Time("now", now),
			zap.Time("last_enable", lastEnable),
			zap.Time("next_disable", nextDisable),
		)

		// Check if 'now' is at or after 'lastEnable' and before 'nextDisable'
		if now.Equal(lastEnable) || (now.After(lastEnable) && now.Before(nextDisable)) {
			cm.logger.Debug("Request matches within the enable/disable range",
				zap.Time("now", now),
				zap.Time("last_enable", lastEnable),
				zap.Time("next_disable", nextDisable),
			)
			return true
		}
	}

	cm.logger.Debug("Request did not match any cron schedule", zap.Time("now", now))
	return false
}

// Interface guards
var (
	_ caddy.Provisioner        = (*CronMatcher)(nil)
	_ caddy.Validator          = (*CronMatcher)(nil)
	_ caddyfile.Unmarshaler    = (*CronMatcher)(nil)
	_ caddyhttp.RequestMatcher = (*CronMatcher)(nil)
)
