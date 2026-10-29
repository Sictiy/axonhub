package biz

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent/apikey"
	"github.com/looplj/axonhub/internal/ent/privacy"
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
	Models  []UserUsageBreakdown `json:"models"`
	APIKeys []UserUsageBreakdown `json:"apiKeys"`
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
	ctx = privacy.DecisionContext(ctx, privacy.Allow)
	result := &UserUsageDetails{Models: []UserUsageBreakdown{}, APIKeys: []UserUsageBreakdown{}}
	for _, byModel := range []bool{true, false} {
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
			sel.Where(sql.And(sql.EQ(keys.C(apikey.FieldUserID), userID), sql.EQ(keys.C(apikey.FieldDeletedAt), 0)))
			id, name := keys.C(apikey.FieldID), keys.C(apikey.FieldName)
			if byModel {
				id, name = sel.C(usagelog.FieldModelID), sel.C(usagelog.FieldModelID)
			}
			sel.Select(sql.As(id, "id"), sql.As(name, "name"), sql.As(sql.Count(sel.C(usagelog.FieldID)), "request_count"), sql.As(fmt.Sprintf("COALESCE(SUM(%s), 0)", sel.C(usagelog.FieldTotalTokens)), "total_tokens"), sql.As(fmt.Sprintf("COALESCE(SUM(%s), 0)", sel.C(usagelog.FieldTotalCost)), "total_cost")).GroupBy(id, name).OrderBy(sql.Desc("request_count"))
		}).Scan(ctx, &rows)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			v := UserUsageBreakdown{ID: row.ID, Name: row.Name, RequestCount: row.Count, TotalTokens: row.Tokens, TotalCost: row.Cost}
			if byModel {
				result.Models = append(result.Models, v)
			} else {
				result.APIKeys = append(result.APIKeys, v)
			}
		}
	}
	return result, nil
}
