package db

// QueryLatencyStats returns current per-query latency distribution samples.
func (c *Database) QueryLatencyStats() []queryLatencyStats {
	if c == nil || c.tracker == nil {
		return nil
	}
	return c.tracker.snapshot()
}
