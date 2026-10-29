package biz

import (
	"context"
	"testing"
	"time"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/enttest"
	"github.com/stretchr/testify/require"
)

func TestUserUsageDetailsProjectUserAndTime(t *testing.T) {
	client := enttest.NewEntClient(t, "sqlite3", "file:usage-details-test?mode=memory&_fk=0")
	defer client.Close()
	ctx := authz.WithTestBypass(ent.NewContext(context.Background(), client))
	ctx = contexts.WithProjectID(ctx, 1)
	ctx = contexts.WithUser(ctx, &ent.User{IsOwner: true})
	service := &SystemService{}
	key1, err := client.APIKey.Create().SetName("first").SetKey("key-1").SetUserID(10).SetProjectID(1).Save(ctx)
	require.NoError(t, err)
	key2, err := client.APIKey.Create().SetName("second").SetKey("key-2").SetUserID(10).SetProjectID(1).Save(ctx)
	require.NoError(t, err)
	other, err := client.APIKey.Create().SetName("other").SetKey("key-3").SetUserID(11).SetProjectID(1).Save(ctx)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	for i, v := range []struct {
		key, project int
		model        string
		at           time.Time
		tokens       int64
	}{{key1.ID, 1, "model-a", now, 10}, {key2.ID, 1, "model-a", now, 20}, {key1.ID, 1, "model-b", now, 30}, {other.ID, 1, "model-a", now, 40}, {key1.ID, 2, "model-a", now, 50}, {key1.ID, 1, "model-a", now.Add(-time.Hour), 60}} {
		_, err = client.UsageLog.Create().SetRequestID(i + 1).SetAPIKeyID(v.key).SetProjectID(v.project).SetModelID(v.model).SetTotalTokens(v.tokens).SetCreatedAt(v.at).Save(ctx)
		require.NoError(t, err)
	}
	from := now.Add(-time.Minute)
	to := now.Add(time.Minute)
	data, err := service.UserUsageDetails(ctx, 10, &from, &to)
	require.NoError(t, err)
	require.Len(t, data.Models, 2)
	require.Len(t, data.APIKeys, 2)
	require.Equal(t, "model-a", data.Models[0].Name)
	require.Equal(t, 2, data.Models[0].RequestCount)
	require.EqualValues(t, 30, data.Models[0].TotalTokens)
	require.Equal(t, "first", data.APIKeys[0].Name)
	require.EqualValues(t, 40, data.APIKeys[0].TotalTokens)
	_, err = service.UserUsageDetails(contexts.WithUser(ctx, &ent.User{}), 10, &from, &to)
	require.Error(t, err)
}
