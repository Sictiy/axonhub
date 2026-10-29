import { ensureFreshAccessToken } from '@/lib/auth-session';
import { useQuery } from '@tanstack/react-query';
import { graphqlRequest } from '@/gql/graphql';
import { useSelectedProjectId } from '@/stores/projectStore';
import { usageStatsByUserSchema, type UsageStatsByUser } from '@/features/dashboard/data/dashboard';

const USAGE_STATS_BY_USER_QUERY = `
  query GetUsageStatsByUser($timeWindow: String) {
    usageStatsByUser(timeWindow: $timeWindow) {
      userId
      userName
      requestCount
      totalTokens
      totalCost
    }
  }
`;

export function useUsageStatsByUser(timeWindow?: string) {
  const selectedProjectId = useSelectedProjectId();

  return useQuery({
    queryKey: ['usageStatsByUser', timeWindow, selectedProjectId],
    queryFn: async () => {
      const headers = selectedProjectId ? { 'X-Project-ID': selectedProjectId } : undefined;
      const data = await graphqlRequest<{ usageStatsByUser: UsageStatsByUser[] }>(
        USAGE_STATS_BY_USER_QUERY,
        { timeWindow },
        headers
      );
      return data.usageStatsByUser.map((item) => usageStatsByUserSchema.parse(item));
    },
    enabled: !!selectedProjectId,
    refetchInterval: 60000,
    placeholderData: (previousData) => previousData,
  });
}


type UsageBreakdown = { id: string; name: string; requestCount: number; totalTokens: number; totalCost: number };
type UserUsageDetails = { models: UsageBreakdown[]; apiKeys: UsageBreakdown[] };

export function useUserUsageDetails(userId?: string, from?: string, to?: string) {
  const projectId = useSelectedProjectId();
  return useQuery({
    queryKey: ['userUsageDetails', projectId, userId, from, to],
    enabled: !!projectId && !!userId,
    queryFn: async ({ signal }): Promise<UserUsageDetails> => {
      const token = await ensureFreshAccessToken();
      const params = new URLSearchParams({ userId: userId! });
      if (from) params.set('from', from);
      if (to) params.set('to', to);
      const response = await fetch(`/admin/system/user-usage-details?${params}`, {
        headers: { 'X-Project-ID': projectId!, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        signal,
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      return response.json();
    },
  });
}
