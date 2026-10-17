package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseReplicaURLs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace", "   ", nil},
		{"single", "postgres://a", []string{"postgres://a"}},
		{"multiple", "postgres://a, postgres://b ,postgres://c", []string{"postgres://a", "postgres://b", "postgres://c"}},
		{"trailing comma", "postgres://a,", []string{"postgres://a"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseReplicaURLs(tc.in))
		})
	}
}

func TestLoadReplicaConfig(t *testing.T) {
	t.Setenv("DATABASE_REPLICA_URLS", "postgres://replica-1, postgres://replica-2")
	t.Setenv("DB_REPLICA_LAG_THRESHOLD", "45s")
	t.Setenv("DB_REPLICA_HEALTH_INTERVAL", "20s")
	t.Setenv("DB_REPLICA_FAILOVER_TO_PRIMARY", "false")
	t.Setenv("DB_REPLICA_MAX_IDLE_TIME", "2m")
	t.Setenv("DB_REPLICA_MAX_LIFETIME", "1h")

	cfg := LoadReplicaConfig()

	assert.True(t, cfg.Enabled())
	assert.Equal(t, []string{"postgres://replica-1", "postgres://replica-2"}, cfg.URLs)
	assert.Equal(t, 45*time.Second, cfg.LagThreshold)
	assert.Equal(t, 20*time.Second, cfg.HealthInterval)
	assert.False(t, cfg.FailoverToPrimary)
	assert.Equal(t, 2*time.Minute, cfg.MaxIdleReplicaAge)
	assert.Equal(t, time.Hour, cfg.MaxReplicaLifetime)
}

func TestLoadReplicaConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_REPLICA_URLS", "")
	t.Setenv("DB_REPLICA_LAG_THRESHOLD", "")
	t.Setenv("DB_REPLICA_HEALTH_INTERVAL", "")
	t.Setenv("DB_REPLICA_FAILOVER_TO_PRIMARY", "")

	cfg := LoadReplicaConfig()

	assert.False(t, cfg.Enabled())
	assert.Equal(t, 30*time.Second, cfg.LagThreshold)
	assert.Equal(t, 15*time.Second, cfg.HealthInterval)
	assert.True(t, cfg.FailoverToPrimary)
}

func TestDBPoolConfigValidation(t *testing.T) {
	valid := DBPoolConfig{
		MaxOpen:      25,
		MaxIdle:      10,
		MaxLifetime:  30 * time.Minute,
		MaxIdleTime:  5 * time.Minute,
		ConnectRetry: 3,
		ConnectWait:  time.Second,
	}
	assert.NoError(t, valid.Validate())

	invalidOpen := valid
	invalidOpen.MaxOpen = 0
	assert.Error(t, invalidOpen.Validate())

	invalidIdle := valid
	invalidIdle.MaxIdle = -1
	assert.Error(t, invalidIdle.Validate())

	invalidRetry := valid
	invalidRetry.ConnectRetry = -1
	assert.Error(t, invalidRetry.Validate())
}

func TestDBPoolConfigWarnings(t *testing.T) {
	// 1. MaxIdle > MaxOpen
	cfgIdleHigher := DBPoolConfig{
		MaxOpen: 10,
		MaxIdle: 20,
	}
	warnings := cfgIdleHigher.CheckWarnings()
	assert.NotEmpty(t, warnings)
	assert.Equal(t, 10, cfgIdleHigher.MaxIdle, "MaxIdle should be capped to MaxOpen")

	// 2. MaxOpen > 100
	cfgHighOpen := DBPoolConfig{
		MaxOpen: 150,
		MaxIdle: 10,
	}
	warnings = cfgHighOpen.CheckWarnings()
	assert.Contains(t, warnings[0], "is high")

	// 3. MaxOpen < 5
	cfgLowOpen := DBPoolConfig{
		MaxOpen: 3,
		MaxIdle: 2,
	}
	warnings = cfgLowOpen.CheckWarnings()
	assert.Contains(t, warnings[0], "is very low")

	// 4. MaxLifetime == 0
	cfgZeroLifetime := DBPoolConfig{
		MaxOpen:     25,
		MaxIdle:     10,
		MaxLifetime: 0,
	}
	warnings = cfgZeroLifetime.CheckWarnings()
	assert.Contains(t, warnings[0], "DB_CONN_MAX_LIFETIME is 0")

	// 5. MaxIdleTime > MaxLifetime
	cfgIdleExceedsLifetime := DBPoolConfig{
		MaxOpen:     25,
		MaxIdle:     10,
		MaxLifetime: 5 * time.Minute,
		MaxIdleTime: 10 * time.Minute,
	}
	warnings = cfgIdleExceedsLifetime.CheckWarnings()
	assert.Contains(t, warnings[0], "greater than DB_CONN_MAX_LIFETIME")
}

func TestLoadDBPoolConfigEnvironment(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "25")
	t.Setenv("DB_CONN_MAX_LIFETIME", "45m")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "10m")
	t.Setenv("DB_CONNECT_RETRIES", "5")
	t.Setenv("DB_CONNECT_RETRY_WAIT", "2s")

	cfg := LoadDBPoolConfig()
	assert.Equal(t, 50, cfg.MaxOpen)
	assert.Equal(t, 25, cfg.MaxIdle)
	assert.Equal(t, 45*time.Minute, cfg.MaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.MaxIdleTime)
	assert.Equal(t, 5, cfg.ConnectRetry)
	assert.Equal(t, 2*time.Second, cfg.ConnectWait)
}

