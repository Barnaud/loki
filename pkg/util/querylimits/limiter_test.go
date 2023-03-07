package querylimits

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/loki/pkg/validation"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

// copied from loki oss
type mockTenantLimits struct {
	limits map[string]*validation.Limits
}

func newMockTenantLimits(limits map[string]*validation.Limits) *mockTenantLimits {
	return &mockTenantLimits{
		limits: limits,
	}
}

func (l *mockTenantLimits) TenantLimits(userID string) *validation.Limits {
	return l.limits[userID]
}

func (l *mockTenantLimits) AllByUserID() map[string]*validation.Limits { return l.limits }

// end copy pasta

func TestLimiter_Defaults(t *testing.T) {
	// some fake tenant
	tLimits := make(map[string]*validation.Limits)
	tLimits["fake"] = &validation.Limits{
		QueryTimeout:            model.Duration(30 * time.Second),
		MaxQueryLookback:        model.Duration(30 * time.Second),
		MaxQueryLength:          model.Duration(30 * time.Second),
		MaxEntriesLimitPerQuery: 10,
	}

	overrides, _ := validation.NewOverrides(validation.Limits{}, newMockTenantLimits(tLimits))
	l := NewLimiter(overrides)

	expectedLimits := QueryLimits{
		MaxQueryLength:          model.Duration(30 * time.Second),
		MaxQueryLookback:        model.Duration(30 * time.Second),
		MaxEntriesLimitPerQuery: 10,
		QueryTimeout:            model.Duration(30 * time.Second),
	}
	ctx := context.Background()
	queryLookback, err := l.MaxQueryLookback(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(expectedLimits.MaxQueryLookback), queryLookback)
	queryLength, err := l.MaxQueryLength(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(expectedLimits.MaxQueryLength), queryLength)
	maxEntries, err := l.MaxEntriesLimitPerQuery(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, expectedLimits.MaxEntriesLimitPerQuery, maxEntries)
	queryTimeout, err := l.QueryTimeout(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(expectedLimits.QueryTimeout), queryTimeout)

	var limits QueryLimits
	limits.QueryTimeout = model.Duration(29 * time.Second)

	expectedLimits2 := QueryLimits{
		MaxQueryLength:          model.Duration(30 * time.Second),
		MaxQueryLookback:        model.Duration(30 * time.Second),
		MaxEntriesLimitPerQuery: 10,
		QueryTimeout:            model.Duration(29 * time.Second),
	}
	{
		ctx2 := InjectQueryLimitsContext(context.Background(), limits)
		queryLookback, err := l.MaxQueryLookback(ctx2, "fake")
		require.NoError(t, err)
		require.Equal(t, time.Duration(expectedLimits2.MaxQueryLookback), queryLookback)
		queryLength, err := l.MaxQueryLength(ctx2, "fake")
		require.NoError(t, err)
		require.Equal(t, time.Duration(expectedLimits2.MaxQueryLength), queryLength)
		maxEntries, err := l.MaxEntriesLimitPerQuery(ctx2, "fake")
		require.NoError(t, err)
		require.Equal(t, expectedLimits2.MaxEntriesLimitPerQuery, maxEntries)
		queryTimeout, err := l.QueryTimeout(ctx2, "fake")
		require.NoError(t, err)
		require.Equal(t, time.Duration(expectedLimits2.QueryTimeout), queryTimeout)
	}

}

func TestLimiter_RejectHighLimits(t *testing.T) {
	// some fake tenant
	tLimits := make(map[string]*validation.Limits)
	tLimits["fake"] = &validation.Limits{
		QueryTimeout:            model.Duration(30 * time.Second),
		MaxQueryLookback:        model.Duration(30 * time.Second),
		MaxQueryLength:          model.Duration(30 * time.Second),
		MaxEntriesLimitPerQuery: 10,
	}

	overrides, _ := validation.NewOverrides(validation.Limits{}, newMockTenantLimits(tLimits))
	l := NewLimiter(overrides)
	limits := QueryLimits{
		MaxQueryLength:          model.Duration(2 * 24 * time.Hour),
		MaxQueryLookback:        model.Duration(14 * 24 * time.Hour),
		MaxEntriesLimitPerQuery: 100,
		QueryTimeout:            model.Duration(100 * time.Second),
	}

	ctx := InjectQueryLimitsContext(context.Background(), limits)
	_, err := l.MaxQueryLookback(ctx, "fake")
	require.Error(t, err)
	_, err = l.MaxQueryLength(ctx, "fake")
	require.Error(t, err)
	_, err = l.MaxEntriesLimitPerQuery(ctx, "fake")
	require.Error(t, err)
	_, err = l.QueryTimeout(ctx, "fake")
	require.Error(t, err)
}

func TestLimiter_AcceptLowerLimits(t *testing.T) {
	// some fake tenant
	tLimits := make(map[string]*validation.Limits)
	tLimits["fake"] = &validation.Limits{
		QueryTimeout:            model.Duration(30 * time.Second),
		MaxQueryLookback:        model.Duration(30 * time.Second),
		MaxQueryLength:          model.Duration(30 * time.Second),
		MaxEntriesLimitPerQuery: 10,
	}

	overrides, _ := validation.NewOverrides(validation.Limits{}, newMockTenantLimits(tLimits))
	l := NewLimiter(overrides)
	limits := QueryLimits{
		MaxQueryLength:          model.Duration(29 * time.Second),
		MaxQueryLookback:        model.Duration(29 * time.Second),
		MaxEntriesLimitPerQuery: 9,
		QueryTimeout:            model.Duration(29 * time.Second),
	}

	ctx := InjectQueryLimitsContext(context.Background(), limits)
	queryLookback, err := l.MaxQueryLookback(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(limits.MaxQueryLookback), queryLookback)
	queryLength, err := l.MaxQueryLength(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(limits.MaxQueryLength), queryLength)
	maxEntries, err := l.MaxEntriesLimitPerQuery(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, limits.MaxEntriesLimitPerQuery, maxEntries)
	queryTimeout, err := l.QueryTimeout(ctx, "fake")
	require.NoError(t, err)
	require.Equal(t, time.Duration(limits.QueryTimeout), queryTimeout)
}