import assert from 'node:assert/strict';
import test from 'node:test';
import { buildItemsFromProviderModel } from './context-price.ts';

test('catalog context tiers apply to input, cached input and output using the same threshold', () => {
  const result = buildItemsFromProviderModel({ cost: {
    input: 2.5, output: 15, cache_read: 0.25,
    tiers: [{ tier: { type: 'context', size: 272000 }, input: 5, output: 22.5, cache_read: 0.5 }],
  } });
  assert.deepEqual(result.map(({ itemCode, pricing }) => [itemCode, pricing.mode, pricing.usageTiered.tiers]), [
    ['prompt_tokens', 'input_context_volume', [{ upTo: 272000, pricePerUnit: '2.5000' }, { upTo: null, pricePerUnit: '5.0000' }]],
    ['completion_tokens', 'input_context_volume', [{ upTo: 272000, pricePerUnit: '15.0000' }, { upTo: null, pricePerUnit: '22.5000' }]],
    ['prompt_cached_tokens', 'input_context_volume', [{ upTo: 272000, pricePerUnit: '0.2500' }, { upTo: null, pricePerUnit: '0.5000' }]],
  ]);
});

test('ordinary rates remain per-unit and non-context tiers do not trigger context pricing', () => {
  const [item] = buildItemsFromProviderModel({ cost: { input: 1, tiers: [{ tier: { type: 'other', size: 50 }, input: 9 }] } });
  assert.deepEqual(item.pricing, { mode: 'usage_per_unit', usagePerUnit: '1.0000' });
});

test('context tiers are ordered and per-item price gaps keep the last known rate', () => {
  const [item] = buildItemsFromProviderModel({ cost: { input: 1, tiers: [
    { tier: { type: 'context', size: 300 }, input: 3 },
    { tier: { type: 'context', size: 100 }, input: 2 },
    { tier: { type: 'context', size: 200 }, output: 7 },
  ] } });
  assert.deepEqual(item.pricing.usageTiered.tiers, [
    { upTo: 100, pricePerUnit: '1.0000' },
    { upTo: 300, pricePerUnit: '2.0000' },
    { upTo: null, pricePerUnit: '3.0000' },
  ]);
});
