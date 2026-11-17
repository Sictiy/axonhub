// 职责: 验证长上下文价格模式可通过 GraphQL 输入输出。
// 关联文件:
//
//	internal/server/gql/price.graphql
//	internal/objects/price.go
package gql

import (
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"testing"
)

func TestInputContextPricingModeGraphQLSchema(t *testing.T) {
	schema := NewExecutableSchema(Config{Resolvers: &Resolver{}}).Schema()
	require.NotNil(t, schema.Types["PricingMode"].EnumValues.ForName("input_context_volume"))
	_, err := gqlparser.LoadQuery(schema, `mutation { saveChannelModelPrices(channelId: "gid://axonhub/Channel/1", input: [{modelId: "m", price: {items: [{itemCode: prompt_tokens, pricing: {mode: input_context_volume, usageTiered: {tiers: [{upTo: 272000, pricePerUnit: "1"}, {pricePerUnit: "2"}]}}}]}}]) { id } }`)
	require.Empty(t, err)
}
