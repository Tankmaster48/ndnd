package congestion

// FixedCongestionControl is an implementation of CongestionWindow with a fixed window size that does not change in response to signals or events.
type FixedCongestionWindow struct {
	window int // window size
}

// Constructs a new FixedCongestionWindow with the specified congestion window size.
func NewFixedCongestionWindow(cwnd int) *FixedCongestionWindow {
	return &FixedCongestionWindow{
		window: cwnd,
	}
}

// log identifier
func (cw *FixedCongestionWindow) String() string {
	return "fixed-congestion-window"
}

// Returns the current size of the fixed congestion window as an integer.
func (cw *FixedCongestionWindow) Size() int {
	return cw.window
}

// No-ops (does nothing) because the congestion window size is fixed and cannot be increased.
func (cw *FixedCongestionWindow) IncreaseWindow() {
	// intentionally left blank: window size is fixed
}

// This method is intended to decrease the congestion window size, but has no effect as the window size is fixed.
func (cw *FixedCongestionWindow) DecreaseWindow() {
	// intentionally left blank: window size is fixed
}

// Handles congestion signals by taking no action, as the fixed congestion window size remains constant regardless of network conditions.
func (cw *FixedCongestionWindow) HandleSignal(signal CongestionSignal) {
	// intentionally left blank: fixed CW doesn't respond to signals
}
