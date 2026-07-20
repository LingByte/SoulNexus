package common

const (
	DefaultSessionIdleTimeoutHours = 12
	DefaultSessionMaxLifetimeHours = 48
	MinSessionIdleTimeoutHours     = 1
	MaxSessionIdleTimeoutHours     = 168
	MinSessionMaxLifetimeHours     = 1
	MaxSessionMaxLifetimeHours     = 720
	TrustDeviceLoginDays           = 7
)

func normalizeSessionIdleHours(v int) int {
	if v <= MinSessionIdleTimeoutHours {
		return DefaultSessionIdleTimeoutHours
	}
	if v > MaxSessionIdleTimeoutHours {
		return MaxSessionIdleTimeoutHours
	}
	return v
}

func normalizeSessionMaxLifetimeHours(idle, max int) int {
	idle = normalizeSessionIdleHours(idle)
	max = normalizeSessionMaxLifetimeHoursOnly(max)
	if max < idle {
		return idle
	}
	return max
}

func normalizeSessionMaxLifetimeHoursOnly(v int) int {
	if v <= 0 {
		return DefaultSessionMaxLifetimeHours
	}
	if v < MinSessionMaxLifetimeHours {
		return MinSessionMaxLifetimeHours
	}
	if v > MaxSessionMaxLifetimeHours {
		return MaxSessionMaxLifetimeHours
	}
	return v
}

// SessionIdleTimeout clamps idle session hours into the allowed range.
func SessionIdleTimeout(idleHours int) int {
	return normalizeSessionIdleHours(idleHours)
}

// SessionMaxLifetime clamps max session lifetime and ensures it is not shorter than idle timeout.
func SessionMaxLifetime(idleHours, maxHours int) int {
	return normalizeSessionMaxLifetimeHours(idleHours, maxHours)
}
