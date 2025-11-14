// Package behavior provides behavior simulation.
// This file simulates server load effects on performance.
package behavior

import (
	"sync/atomic"
	"time"
)

// LoadSimulator tracks and simulates server load effects.
// High load causes slower responses in production.
type LoadSimulator struct {
	// currentRPS tracks current requests per second
	currentRPS atomic.Int64

	// window is the time window for RPS calculation
	window time.Duration

	// requestTimestamps stores recent request timestamps
	timestamps []time.Time
	timestampsIdx int

	// loadThresholds define load levels
	lowLoad    int64
	mediumLoad int64
	highLoad   int64
}

// NewLoadSimulator creates a new load simulator.
func NewLoadSimulator() *LoadSimulator {
	return &LoadSimulator{
		window:     10 * time.Second,
		timestamps: make([]time.Time, 1000), // Track last 1000 requests
		lowLoad:    10,
		mediumLoad: 50,
		highLoad:   100,
	}
}

// RecordRequest records a request for load tracking.
func (ls *LoadSimulator) RecordRequest() {
	now := time.Now()
	
	// Store timestamp
	ls.timestamps[ls.timestampsIdx] = now
	ls.timestampsIdx = (ls.timestampsIdx + 1) % len(ls.timestamps)
	
	// Calculate current RPS
	rps := ls.calculateRPS(now)
	ls.currentRPS.Store(rps)
}

// calculateRPS calculates requests per second in the current window.
func (ls *LoadSimulator) calculateRPS(now time.Time) int64 {
	cutoff := now.Add(-ls.window)
	count := int64(0)
	
	for _, ts := range ls.timestamps {
		if !ts.IsZero() && ts.After(cutoff) {
			count++
		}
	}
	
	seconds := ls.window.Seconds()
	if seconds > 0 {
		return int64(float64(count) / seconds)
	}
	
	return 0
}

// GetCurrentRPS returns the current requests per second.
func (ls *LoadSimulator) GetCurrentRPS() int64 {
	return ls.currentRPS.Load()
}

// GetLoadLevel returns the current load level.
func (ls *LoadSimulator) GetLoadLevel() LoadLevel {
	rps := ls.GetCurrentRPS()
	
	if rps < ls.lowLoad {
		return LoadIdle
	} else if rps < ls.mediumLoad {
		return LoadLow
	} else if rps < ls.highLoad {
		return LoadMedium
	}
	
	return LoadHigh
}

// IsHighLoad returns true if load is considered high.
func (ls *LoadSimulator) IsHighLoad() bool {
	return ls.GetCurrentRPS() >= ls.highLoad
}

// LoadLevel represents server load levels.
type LoadLevel string

const (
	LoadIdle   LoadLevel = "idle"
	LoadLow    LoadLevel = "low"
	LoadMedium LoadLevel = "medium"
	LoadHigh   LoadLevel = "high"
)