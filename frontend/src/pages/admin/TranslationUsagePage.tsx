import React, { useState, useEffect } from 'react';
import {
  Container,
  Title,
  Card,
  Text,
  Stack,
  Group,
  Badge,
  Progress,
  SimpleGrid,
  Tabs,
  Table,
  RingProgress,
  Alert,
  Loader,
  Modal,
  Button,
} from '@mantine/core';
import {
  IconAlertCircle,
  IconInfoCircle,
} from '@tabler/icons-react';
import * as TablerIcons from '@tabler/icons-react';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<React.SVGProps<SVGSVGElement> & { size?: number }>
>;
const IconArrowUp: React.ComponentType<React.SVGProps<SVGSVGElement> & { size?: number }> =
  tablerIconMap.IconArrowUp || (() => null);
const IconChartLine: React.ComponentType<React.SVGProps<SVGSVGElement> & { size?: number }> =
  tablerIconMap.IconChartLine || (() => null);
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { adminApi } from '../../api/admin';

interface UsageStats {
  usage_stats: {
    [serviceName: string]: {
      [month: string]: {
        [usageType: string]: {
          characters_used: number;
          requests_made: number;
          quota: number;
        };
      };
    };
  };
  monthly_totals: {
    [month: string]: {
      [serviceName: string]: {
        total_characters: number;
        total_requests: number;
      };
    };
  };
  services: string[];
  cache_stats?: {
    total_cache_hits_requests: number;
    total_cache_hits_characters: number;
    total_cache_misses_requests: number;
    cache_hit_rate: number;
  };
}

const TranslationUsagePage: React.FC = () => {
  const [usageStats, setUsageStats] = useState<UsageStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [chartModalOpen, setChartModalOpen] = useState(false);

  useEffect(() => {
    loadUsageStats();
  }, []);

  const loadUsageStats = async () => {
    try {
      setLoading(true);
      setError(null);

      // Load overview stats
      const statsResponse = await adminApi.getUsageStats();
      setUsageStats(statsResponse.data);
    } catch (error: unknown) {
      let errorMessage = 'Failed to load usage statistics';

      if (error && typeof error === 'object' && 'response' in error) {
        const responseError = error as {
          response?: { data?: { message?: string } };
        };
        errorMessage = responseError.response?.data?.message || errorMessage;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }

      setError(errorMessage);
      console.error('Error loading usage stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatNumber = (num: number): string => {
    return new Intl.NumberFormat().format(num);
  };

  const formatMonth = (monthStr: string): string => {
    const [year, month] = monthStr.split('-');
    return new Date(parseInt(year), parseInt(month) - 1).toLocaleDateString(
      'en-US',
      {
        year: 'numeric',
        month: 'short',
      }
    );
  };

  const prepareChartData = () => {
    if (!usageStats) return [];

    const chartData: Array<{
      month: string;
      monthKey: string;
      [key: string]: string | number;
    }> = [];

    // Build a map to track cache stats by month
    const cacheDataByMonth: {
      [month: string]: {
        cache_hits: number;
        cache_misses: number;
        cache_hit_rate: number;
      };
    } = {};

    // Process cache stats from usage_stats
    usageStats.services.forEach(service => {
      const serviceData = usageStats.usage_stats[service];
      if (serviceData) {
        Object.entries(serviceData).forEach(([month, usageTypes]) => {
          if (!cacheDataByMonth[month]) {
            cacheDataByMonth[month] = {
              cache_hits: 0,
              cache_misses: 0,
              cache_hit_rate: 0,
            };
          }

          if (usageTypes['translation_cache_hit']) {
            cacheDataByMonth[month].cache_hits +=
              usageTypes['translation_cache_hit'].requests_made || 0;
          }
          if (usageTypes['translation_cache_miss']) {
            cacheDataByMonth[month].cache_misses +=
              usageTypes['translation_cache_miss'].requests_made || 0;
          }

          // Calculate hit rate for this month
          const totalRequests =
            cacheDataByMonth[month].cache_hits +
            cacheDataByMonth[month].cache_misses;
          if (totalRequests > 0) {
            cacheDataByMonth[month].cache_hit_rate =
              (cacheDataByMonth[month].cache_hits / totalRequests) * 100;
          }
        });
      }
    });

    // Process monthly totals for all services
    Object.entries(usageStats.monthly_totals)
      .sort(([a], [b]) => a.localeCompare(b))
      .forEach(([month, services]) => {
        const monthData: Record<string, number | string> = {
          month: formatMonth(month),
          monthKey: month, // Keep original for reference
        };

        Object.entries(services).forEach(([service, data]) => {
          monthData[`${service}_characters`] = data.total_characters;
          monthData[`${service}_requests`] = data.total_requests;
        });

        // Add cache data if available for this month
        if (cacheDataByMonth[month]) {
          monthData['cache_hits'] = cacheDataByMonth[month].cache_hits;
          monthData['cache_misses'] = cacheDataByMonth[month].cache_misses;
          monthData['cache_hit_rate'] = cacheDataByMonth[month].cache_hit_rate;
        }

        chartData.push(monthData as { [key: string]: number | string; month: string; monthKey: string });
      });

    return chartData;
  };

  const getQuotaPercentage = (used: number, quota: number): number => {
    if (quota === 0) return 0;
    return Math.min((used / quota) * 100, 100);
  };

  const getQuotaColor = (percentage: number): string => {
    if (percentage >= 90) return 'red';
    if (percentage >= 75) return 'orange';
    if (percentage >= 50) return 'yellow';
    return 'green';
  };

  if (loading) {
    return (
      <Container size='lg' py='xl'>
        <Stack align='center' gap='md'>
          <Loader size='lg' />
          <Text>Loading usage statistics...</Text>
        </Stack>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size='lg' py='xl'>
        <Alert
          variant='light'
          color='red'
          title='Error'
          icon={<IconAlertCircle size={16} />}
        >
          {error}
        </Alert>
      </Container>
    );
  }

  if (!usageStats) {
    return (
      <Container size='lg' py='xl'>
        <Alert
          variant='light'
          color='blue'
          title='No Data'
          icon={<IconInfoCircle size={16} />}
        >
          No usage statistics available yet. Translation usage will appear here
          once translations are performed.
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='lg' py='xl'>
      <Stack gap='xl'>
        <div>
          <Title order={1} mb='xs'>
            Translation Usage Statistics
          </Title>
          <Text color='dimmed' size='lg'>
            Monitor translation service usage and quotas across all services.
          </Text>
        </div>

        <SimpleGrid cols={{ base: 1, md: 2, lg: 4 }} spacing='md'>
          {usageStats.services.map(service => {
            const latestMonth = Object.keys(
              usageStats.usage_stats[service] || {}
            )
              .sort()
              .pop();
            if (!latestMonth) return null;

            const latestData = usageStats.usage_stats[service][latestMonth];
            // Only count translation usage for quota, not cache
            const translationUsage = latestData['translation'];
            const totalUsed = translationUsage?.characters_used || 0;
            const totalQuota = translationUsage?.quota || 0;

            const percentage = getQuotaPercentage(totalUsed, totalQuota);

            return (
              <Card
                key={service}
                shadow='sm'
                padding='lg'
                radius='md'
                withBorder
              >
                <Stack gap='xs'>
                  <Group justify='space-between'>
                    <Text fw={500} size='sm' tt='uppercase' c='dimmed'>
                      {service.toUpperCase()}
                    </Text>
                    <Badge color={getQuotaColor(percentage)} variant='light'>
                      {percentage.toFixed(1)}%
                    </Badge>
                  </Group>

                  <div>
                    <Text size='xl' fw={700}>
                      {formatNumber(totalUsed)}
                    </Text>
                    <Text size='sm' c='dimmed'>
                      of {formatNumber(totalQuota)} characters
                    </Text>
                  </div>

                  <Progress
                    value={percentage}
                    color={getQuotaColor(percentage)}
                    size='sm'
                    striped
                    animated={percentage > 80}
                  />
                </Stack>
              </Card>
            );
          })}

          {/* Cache Statistics Cards */}
          {usageStats.cache_stats && (
            <>
              {/* Cache Hit Rate Card */}
              <Card shadow='sm' padding='lg' radius='md' withBorder>
                <Stack gap='xs'>
                  <Group justify='space-between'>
                    <Text fw={500} size='sm' tt='uppercase' c='dimmed'>
                      Cache Hit Rate
                    </Text>
                    <Badge
                      color={
                        usageStats.cache_stats.cache_hit_rate >= 70
                          ? 'green'
                          : usageStats.cache_stats.cache_hit_rate >= 40
                            ? 'yellow'
                            : 'red'
                      }
                      variant='light'
                    >
                      {usageStats.cache_stats.cache_hit_rate.toFixed(1)}%
                    </Badge>
                  </Group>

                  <div>
                    <RingProgress
                      size={80}
                      thickness={8}
                      sections={[
                        {
                          value: usageStats.cache_stats.cache_hit_rate,
                          color:
                            usageStats.cache_stats.cache_hit_rate >= 70
                              ? 'green'
                              : usageStats.cache_stats.cache_hit_rate >= 40
                                ? 'yellow'
                                : 'red',
                        },
                      ]}
                      label={
                        <Text size='xs' ta='center' fw={700}>
                          {usageStats.cache_stats.cache_hit_rate.toFixed(0)}%
                        </Text>
                      }
                    />
                  </div>

                  <Text size='sm' c='dimmed'>
                    {formatNumber(
                      usageStats.cache_stats.total_cache_hits_requests
                    )}{' '}
                    hits /{' '}
                    {formatNumber(
                      usageStats.cache_stats.total_cache_hits_requests +
                        usageStats.cache_stats.total_cache_misses_requests
                    )}{' '}
                    total
                  </Text>
                </Stack>
              </Card>

              {/* Cache Efficiency Card */}
              <Card shadow='sm' padding='lg' radius='md' withBorder>
                <Stack gap='xs'>
                  <Text fw={500} size='sm' tt='uppercase' c='dimmed'>
                    Cache Efficiency
                  </Text>

                  <div>
                    <Text size='xl' fw={700}>
                      {formatNumber(
                        usageStats.cache_stats.total_cache_hits_requests
                      )}
                    </Text>
                    <Text size='sm' c='dimmed'>
                      requests served from cache
                    </Text>
                  </div>

                  <Text size='xs' c='dimmed'>
                    {formatNumber(
                      usageStats.cache_stats.total_cache_misses_requests
                    )}{' '}
                    cache misses
                  </Text>
                </Stack>
              </Card>

              {/* Characters Saved Card */}
              <Card shadow='sm' padding='lg' radius='md' withBorder>
                <Stack gap='xs'>
                  <Text fw={500} size='sm' tt='uppercase' c='dimmed'>
                    Characters Saved
                  </Text>

                  <div>
                    <Text size='xl' fw={700}>
                      {formatNumber(
                        usageStats.cache_stats.total_cache_hits_characters
                      )}
                    </Text>
                    <Text size='sm' c='dimmed'>
                      characters served from cache
                    </Text>
                  </div>

                  <Text size='xs' c='green'>
                    Quota saved by caching
                  </Text>
                </Stack>
              </Card>
            </>
          )}
        </SimpleGrid>

        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Tabs defaultValue='overview' keepMounted={false}>
              <Tabs.List>
                <Tabs.Tab
                  value='overview'
                  leftSection={<IconArrowUp size={16} />}
                >
                  Overview
                </Tabs.Tab>
                <Tabs.Tab
                  value='monthly'
                  leftSection={<IconArrowUp size={16} />}
                >
                  Monthly Breakdown
                </Tabs.Tab>
              </Tabs.List>

              <Tabs.Panel value='overview' pt='md'>
                <Stack gap='md'>
                  <Text size='lg' fw={500}>
                    Monthly Usage Summary
                  </Text>

                  <Table>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Month</Table.Th>
                        <Table.Th>Service</Table.Th>
                        <Table.Th>Characters Used</Table.Th>
                        <Table.Th>Requests Made</Table.Th>
                        <Table.Th>Quota Usage</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {Object.entries(usageStats.monthly_totals)
                        .sort(([a], [b]) => b.localeCompare(a))
                        .slice(0, 6)
                        .map(([month, services]) =>
                          Object.entries(services).map(([service, data]) => (
                            <Table.Tr key={`${month}-${service}`}>
                              <Table.Td>{formatMonth(month)}</Table.Td>
                              <Table.Td>
                                <Badge variant='light'>{service}</Badge>
                              </Table.Td>
                              <Table.Td>
                                {formatNumber(data.total_characters)}
                              </Table.Td>
                              <Table.Td>
                                {formatNumber(data.total_requests)}
                              </Table.Td>
                              <Table.Td>
                                <Group gap='xs'>
                                  <RingProgress
                                    size={24}
                                    thickness={3}
                                    sections={[
                                      {
                                        value: getQuotaPercentage(
                                          data.total_characters || 0,
                                          500000
                                        ), // Assuming Google quota
                                        color: getQuotaColor(
                                          getQuotaPercentage(
                                            data.total_characters || 0,
                                            500000
                                          )
                                        ),
                                      },
                                    ]}
                                  />
                                  <Text size='sm'>
                                    {getQuotaPercentage(
                                      data.total_characters || 0,
                                      500000
                                    ).toFixed(1)}
                                    %
                                  </Text>
                                </Group>
                              </Table.Td>
                            </Table.Tr>
                          ))
                        )}
                    </Table.Tbody>
                  </Table>
                </Stack>
              </Tabs.Panel>

              <Tabs.Panel value='monthly' pt='md'>
                <Stack gap='md'>
                  <Text size='lg' fw={500}>
                    Service-Specific Usage Details
                  </Text>

                  {usageStats &&
                  Object.keys(usageStats.usage_stats).length > 0 ? (
                    Object.entries(usageStats.usage_stats).map(
                      ([serviceName, serviceData]) => (
                        <Card
                          key={serviceName}
                          shadow='sm'
                          padding='md'
                          radius='md'
                          withBorder
                        >
                          <Text size='md' fw={500} mb='md'>
                            {serviceName.toUpperCase()} Translation Service
                          </Text>

                          <Table>
                            <Table.Thead>
                              <Table.Tr>
                                <Table.Th>Month</Table.Th>
                                <Table.Th>Usage Type</Table.Th>
                                <Table.Th>Characters</Table.Th>
                                <Table.Th>Requests</Table.Th>
                                <Table.Th>Quota</Table.Th>
                                <Table.Th>Usage %</Table.Th>
                              </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                              {Object.entries(serviceData)
                                .sort(([a], [b]) => b.localeCompare(a)) // Sort months descending
                                .map(([month, usageTypes]) =>
                                  Object.entries(usageTypes).map(
                                    ([usageType, data], index) => {
                                      const isCacheType =
                                        usageType === 'translation_cache_hit' ||
                                        usageType === 'translation_cache_miss';
                                      const percentage = getQuotaPercentage(
                                        data.characters_used || 0,
                                        data.quota || 0
                                      );
                                      return (
                                        <Table.Tr
                                          key={`${month}-${usageType}-${index}`}
                                        >
                                          <Table.Td>
                                            {formatMonth(month)}
                                          </Table.Td>
                                          <Table.Td>
                                            <Badge
                                              variant='outline'
                                              size='sm'
                                              color={
                                                isCacheType ? 'blue' : 'green'
                                              }
                                            >
                                              {usageType}
                                            </Badge>
                                          </Table.Td>
                                          <Table.Td>
                                            {formatNumber(
                                              data.characters_used || 0
                                            )}
                                          </Table.Td>
                                          <Table.Td>
                                            {formatNumber(
                                              data.requests_made || 0
                                            )}
                                          </Table.Td>
                                          <Table.Td>
                                            {isCacheType ? (
                                              <Text size='sm' c='dimmed'>
                                                N/A
                                              </Text>
                                            ) : (
                                              formatNumber(data.quota || 0)
                                            )}
                                          </Table.Td>
                                          <Table.Td>
                                            {isCacheType ? (
                                              <Text size='sm' c='dimmed'>
                                                N/A
                                              </Text>
                                            ) : (
                                              <Group gap='xs'>
                                                <Text
                                                  size='sm'
                                                  c={getQuotaColor(percentage)}
                                                >
                                                  {percentage.toFixed(1)}%
                                                </Text>
                                                <Progress
                                                  value={percentage}
                                                  color={getQuotaColor(
                                                    percentage
                                                  )}
                                                  size='xs'
                                                  w={60}
                                                />
                                              </Group>
                                            )}
                                          </Table.Td>
                                        </Table.Tr>
                                      );
                                    }
                                  )
                                )
                                .flat()}
                            </Table.Tbody>
                          </Table>
                        </Card>
                      )
                    )
                  ) : (
                    <Alert
                      variant='light'
                      color='blue'
                      icon={<IconInfoCircle size={16} />}
                    >
                      No detailed service data available yet. This will show
                      per-service usage breakdown once translations are
                      performed.
                    </Alert>
                  )}
                </Stack>
              </Tabs.Panel>
            </Tabs>

            <Button
              leftSection={<IconChartLine size={16} />}
              variant='light'
              onClick={() => setChartModalOpen(true)}
              disabled={
                !usageStats ||
                Object.keys(usageStats.monthly_totals).length === 0
              }
            >
              View Chart
            </Button>
          </Group>
        </Card>
      </Stack>

      <Modal
        opened={chartModalOpen}
        onClose={() => setChartModalOpen(false)}
        title='Usage Statistics Chart'
        size='xl'
      >
        <Stack gap='md'>
          <Text size='sm' c='dimmed'>
            Monthly usage trends across all translation services
          </Text>

          {usageStats && prepareChartData().length > 0 ? (
            <ResponsiveContainer width='100%' height={400}>
              <LineChart data={prepareChartData()}>
                <CartesianGrid strokeDasharray='3 3' />
                <XAxis dataKey='month' />
                <YAxis yAxisId='left' />
                <YAxis
                  yAxisId='right'
                  orientation='right'
                  domain={[0, 100]}
                  label={{
                    value: 'Cache Hit Rate (%)',
                    angle: -90,
                    position: 'insideRight',
                  }}
                />
                <Tooltip
                  formatter={(value: number, name: string) => {
                    if (name.includes('hit_rate')) {
                      return [`${value.toFixed(1)}%`, 'Cache Hit Rate'];
                    }
                    return [
                      formatNumber(value),
                      name.replace(/_/g, ' ').toUpperCase(),
                    ];
                  }}
                />
                <Legend />
                {usageStats.services.map(service => (
                  <Line
                    key={`${service}_characters`}
                    yAxisId='left'
                    type='monotone'
                    dataKey={`${service}_characters`}
                    stroke='#8884d8'
                    name={`${service} Characters`}
                    strokeWidth={2}
                  />
                ))}
                {usageStats.services.map(service => (
                  <Line
                    key={`${service}_requests`}
                    yAxisId='left'
                    type='monotone'
                    dataKey={`${service}_requests`}
                    stroke='#82ca9d'
                    name={`${service} Requests`}
                    strokeWidth={2}
                  />
                ))}
                <Line
                  yAxisId='left'
                  type='monotone'
                  dataKey='cache_hits'
                  stroke='#4dabf7'
                  name='Cache Hits'
                  strokeWidth={2}
                  strokeDasharray='5 5'
                />
                <Line
                  yAxisId='left'
                  type='monotone'
                  dataKey='cache_misses'
                  stroke='#ff6b6b'
                  name='Cache Misses'
                  strokeWidth={2}
                  strokeDasharray='5 5'
                />
                <Line
                  yAxisId='right'
                  type='monotone'
                  dataKey='cache_hit_rate'
                  stroke='#40c057'
                  name='Cache Hit Rate (%)'
                  strokeWidth={3}
                  dot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <Text ta='center' c='dimmed' py='xl'>
              No data available for chart
            </Text>
          )}
        </Stack>
      </Modal>
    </Container>
  );
};

export default TranslationUsagePage;
