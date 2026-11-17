// 职责: 验证按完整输入上下文选择各费用项价格的计费行为。
// 关联文件:
//
//	internal/server/biz/cost_calc.go
//	internal/objects/price.go

package biz

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/llm"
)

func TestInputContextVolumePricesAllItemsByTotalPrompt(t *testing.T) {
	threshold := int64(272000)
	price := func(low, high int64) objects.Pricing {
		return objects.Pricing{Mode: objects.PricingModeInputVolume, UsageTiered: &objects.TieredPricing{Tiers: []objects.PriceTier{{UpTo: &threshold, PricePerUnit: decimal.NewFromInt(low)}, {PricePerUnit: decimal.NewFromInt(high)}}}}
	}
	model := objects.ModelPrice{Items: []objects.ModelPriceItem{
		{ItemCode: objects.PriceItemCodeUsage, Pricing: price(1, 2)},
		{ItemCode: objects.PriceItemCodePromptCachedToken, Pricing: price(1, 2)},
		{ItemCode: objects.PriceItemCodeCompletion, Pricing: price(5, 10)},
	}}
	for _, tc := range []struct {
		prompt   int64
		expected string
	}{{271999, "0.272004"}, {272000, "0.272005"}, {272001, "0.544012"}} {
		t.Run(decimal.NewFromInt(tc.prompt).String(), func(t *testing.T) {
			usage := &llm.Usage{PromptTokens: tc.prompt, CompletionTokens: 1, PromptTokensDetails: &llm.PromptTokensDetails{CachedTokens: 100000}}
			items, total := ComputeUsageCost(usage, model, time.Time{})
			require.True(t, total.Equal(decimal.RequireFromString(tc.expected)), "got %s", total.String())
			require.Len(t, items, 3)
			require.Equal(t, int64(1), items[2].Quantity)
		})
	}
}

func TestInputContextVolumeCacheWriteVariantsAndExistingVolume(t *testing.T) {
	threshold := int64(100)
	pricing := objects.Pricing{Mode: objects.PricingModeInputVolume, UsageTiered: &objects.TieredPricing{Tiers: []objects.PriceTier{{UpTo: &threshold, PricePerUnit: decimal.NewFromInt(1)}, {PricePerUnit: decimal.NewFromInt(3)}}}}
	model := objects.ModelPrice{Items: []objects.ModelPriceItem{{ItemCode: objects.PriceItemCodeWriteCachedTokens, Pricing: pricing, PromptWriteCacheVariants: []objects.PromptWriteCacheVariant{{VariantCode: objects.PromptWriteCacheVariantCode5Min, Pricing: pricing}, {VariantCode: objects.PromptWriteCacheVariantCode1Hour, Pricing: pricing}}}}}
	usage := &llm.Usage{PromptTokens: 101, PromptTokensDetails: &llm.PromptTokensDetails{WriteCachedTokens: 30, WriteCached5MinTokens: 10, WriteCached1HourTokens: 20}}
	items, total := ComputeUsageCost(usage, model, time.Time{})
	require.Len(t, items, 2)
	require.True(t, total.Equal(decimal.RequireFromString("0.00009")), "got %s", total.String())
	legacy := pricing
	legacy.Mode = objects.PricingModeVolume
	item, subtotal := computeItemSubtotalWithInput(10, 101, legacy)
	require.True(t, subtotal.Equal(decimal.RequireFromString("0.00001")), "got %s", subtotal.String())
	require.Equal(t, int64(10), item.Quantity)
}

func TestInputContextVolumePricingValidation(t *testing.T) {
	threshold := int64(272000)
	p := objects.Pricing{Mode: objects.PricingModeInputVolume}
	require.Error(t, p.Validate())
	p.UsageTiered = &objects.TieredPricing{Tiers: []objects.PriceTier{{UpTo: &threshold, PricePerUnit: decimal.NewFromInt(1)}, {PricePerUnit: decimal.NewFromInt(2)}}}
	require.NoError(t, p.Validate())
	require.True(t, p.Equals(&p))
}
