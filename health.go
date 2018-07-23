package main

import (
	"math"
	"runtime"
	"time"
)

var start = time.Now()

// MB helps to represent bytes more user-friendly
const MB float64 = 1.0 * 1024 * 1024

// HealthStats represents current app runtime stats
// swagger:model healthStats
type HealthStats struct {
	Uptime               int64   `json:"uptime"`
	AllocatedMemory      float64 `json:"allocatedMemory"`
	TotalAllocatedMemory float64 `json:"totalAllocatedMemory"`
	Goroutines           int     `json:"goroutines"`
	CPUs                 int     `json:"cpus"`
}

// GetHealthStats returns current app runtime stats
func GetHealthStats() *HealthStats {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	return &HealthStats{
		Uptime:               GetUptime(),
		AllocatedMemory:      toMegaBytes(m.Alloc),
		TotalAllocatedMemory: toMegaBytes(m.TotalAlloc),
		Goroutines:           runtime.NumGoroutine(),
		CPUs:                 runtime.NumCPU(),
	}
}

// GetUptime returns current app uptime
func GetUptime() int64 {
	return time.Now().Unix() - start.Unix()
}

func toMegaBytes(bytes uint64) float64 {
	return toFixed(float64(bytes)/MB, 2)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
