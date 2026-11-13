package metrics

func sloLatencyBuckets() []float64 {
	return []float64{0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2.5, 5}
}
