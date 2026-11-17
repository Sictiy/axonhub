package biz

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent/apikey"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/usagelog"
)

type UserUsageBreakdown struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	RequestCount int     `json:"requestCount"`
	TotalTokens  int64   `json:"totalTokens"`
	TotalCost    float64 `json:"totalCost"`
}

type UserUsageDetails struct {
	Models   []UserUsageBreakdown `json:"models"`
	APIKeys  []UserUsageBreakdown `json:"apiKeys"`
	Channels []UserUsageBreakdown `json:"channels"`
}

func (s *SystemService) UserUsageDetails(ctx context.Context, userID int, from, to *time.Time) (*UserUsageDetails, error) {
	current, ok := contexts.GetUser(ctx)
	if !ok || current == nil {
		return nil, fmt.Errorf("user not found")
	}
	projectID, ok := contexts.GetProjectID(ctx)
	if !ok {
		return nil, fmt.Errorf("project not found")
	}
	owner := current.IsOwner
	for _, p := range current.Edges.ProjectUsers {
		if p.ProjectID == projectID && p.IsOwner {
			owner = true
		}
	}
	if !owner {
		return nil, fmt.Errorf("permission denied")
	}
	return authz.RunAsHost(ctx, s.entFromContext(ctx), "project-user-usage-details", func(ctx context.Context) (*UserUsageDetails, error) {
		result := &UserUsageDetails{Models: []UserUsageBreakdown{}, APIKeys: []UserUsageBreakdown{}, Channels: []UserUsageBreakdown{}}
		for _, grouping := range []string{"model", "key", "channel"} {
			var rows []struct {
				ID     string  `json:"id"`
				Name   string  `json:"name"`
				Count  int     `json:"request_count"`
				Tokens int64   `json:"total_tokens"`
				Cost   float64 `json:"total_cost"`
			}
			q := s.entFromContext(ctx).UsageLog.Query().Where(usagelog.APIKeyIDNotNil(), usagelog.ProjectIDEQ(projectID))
			if from != nil {
				q = q.Where(usagelog.CreatedAtGTE(*from))
			}
			if to != nil {
				q = q.Where(usagelog.CreatedAtLTE(*to))
			}
			err := q.Modify(func(sel *sql.Selector) {
				keys := sql.Table(apikey.Table)
				sel.Join(keys).On(sel.C(usagelog.FieldAPIKeyID), keys.C(apikey.FieldID))
				sel.Where(sql.And(sql.EQ(keys.C(apikey.FieldUserID), userID), sql.EQ(keys.C(apikey.FieldProjectID), projectID)))
				id, name := keys.C(apikey.FieldID), keys.C(apikey.FieldName)
				if grouping == "model" {
					id, name = sel.C(usagelog.FieldModelID), sel.C(usagelog.FieldModelID)
				} else if grouping == "channel" {
					channels := sql.Table(channel.Table)
					sel.LeftJoin(channels).On(sel.C(usagelog.FieldChannelID), channels.C(channel.FieldID))
					id = fmt.Sprintf("COALESCE(%s, 0)", sel.C(usagelog.FieldChannelID))
					name = fmt.Sprintf("COALESCE(%s, '')", channels.C(channel.FieldName))
				}
				sel.Select(sql.As(id, "id"), sql.As(name, "name"), sql.As(sql.Count(sel.C(usagelog.FieldID)), "request_count"), sql.As(fmt.Sprintf("COALESCE(SUM(%s), 0)", sel.C(usagelog.FieldTotalTokens)), "total_tokens"), sql.As(fmt.Sprintf("COALESCE(SUM(%s), 0)", sel.C(usagelog.FieldTotalCost)), "total_cost")).GroupBy(id, name).OrderBy(sql.Desc("request_count"))
			}).Scan(ctx, &rows)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				v := UserUsageBreakdown{ID: row.ID, Name: row.Name, RequestCount: row.Count, TotalTokens: row.Tokens, TotalCost: row.Cost}
				switch grouping {
				case "model":
					result.Models = append(result.Models, v)
				case "key":
					result.APIKeys = append(result.APIKeys, v)
				case "channel":
					result.Channels = append(result.Channels, v)
				}
			}
		}
		return result, nil
	})
}
