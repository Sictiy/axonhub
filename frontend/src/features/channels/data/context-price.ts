// 职责: 将模型目录的价格及上下文分档转换为渠道价格配置。
// 关联文件:
//   frontend/src/features/channels/components/channels-model-price-dialog.tsx
//   frontend/src/features/models/data/providers.schema.ts
import type { ProviderModel } from '@/features/models/data/providers.schema';

type ProviderPriceItemCode = 'prompt_tokens' | 'completion_tokens' | 'prompt_cached_tokens' | 'prompt_write_cached_tokens';
type ProviderPriceItem = {
  itemCode: ProviderPriceItemCode;
  pricing: {
    mode: 'usage_per_unit' | 'input_context_volume';
    usagePerUnit?: string;
    usageTiered?: { tiers: Array<{ upTo: number | null; pricePerUnit: string }> };
  };
};

export function buildItemsFromProviderModel(
  model: ProviderModel,
  multiplier: number = 1,
  exchangeRate: number = 1
): ProviderPriceItem[] {
  const items: ProviderPriceItem[] = [];
  const cost = model.cost;

  const contextTiers = (cost?.tiers ?? [])
    .filter((entry) => entry.tier.type === 'context' && Number.isSafeInteger(entry.tier.size) && (entry.tier.size ?? 0) > 0)
    .sort((a, b) => a.tier.size! - b.tier.size!);
  const format = (value: number) => (value * multiplier * exchangeRate).toFixed(4);
  const addCost = (itemCode: ProviderPriceItemCode, field: 'input' | 'output' | 'cache_read' | 'cache_write') => {
    const base = cost?.[field];
    if (base == null) return;
    const entries = contextTiers.filter((entry) => entry[field] != null);
    if (entries.length === 0) {
      items.push({ itemCode, pricing: { mode: 'usage_per_unit', usagePerUnit: format(base) } });
      return;
    }
    const tiers: Array<{ upTo: number | null; pricePerUnit: string }> = [{ upTo: entries[0].tier.size!, pricePerUnit: format(base) }];
    entries.forEach((entry, index) => tiers.push({
      upTo: entries[index + 1]?.tier.size ?? null,
      pricePerUnit: format(entry[field]!),
    }));
    items.push({ itemCode, pricing: { mode: 'input_context_volume', usageTiered: { tiers } } });
  };
  addCost('prompt_tokens', 'input');
  addCost('completion_tokens', 'output');
  addCost('prompt_cached_tokens', 'cache_read');
  addCost('prompt_write_cached_tokens', 'cache_write');

  if (items.length === 0) {
    items.push({
      itemCode: 'prompt_tokens',
      pricing: { mode: 'usage_per_unit', usagePerUnit: '0' },
    });
  }

  return items;
}

