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
  IconTrendingUp,
  IconAlertTriangle,
  IconInfoCircle,
  IconChartBar,
} from '@tabler/icons-react';
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
}

interface ServiceUsageData {
  service: string;
  data: Array<{
    month: string;
    usage_type: string;
    characters_used: number;
    requests_made: number;
    quota: number;
  }>;
}

const TranslationUsagePage: React.FC = () => {
  const [usageStats, setUsageStats] = useState<UsageStats | null>(null);
  const [serviceUsage, setServiceUsage] = useState<ServiceUsageData | null>(
    null
  );
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

      // Load Google service details for the chart
      try {
        const googleResponse = await adminApi.getUsageStatsByService('google');
        setServiceUsage(googleResponse.data);
      } catch {
        // Google service might not exist yet, that's okay
        console.log('No Google service data available yet');
      }
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

    // Process monthly totals for all services
    Object.entries(usageStats.monthly_totals)
      .sort(([a], [b]) => a.localeCompare(b))
      .forEach(([month, services]) => {
        const monthData = {
          month: formatMonth(month),
          monthKey: month, // Keep original for reference
        };

        Object.entries(services).forEach(([service, data]) => {
          monthData[`${service}_characters`] = data.total_characters;
          monthData[`${service}_requests`] = data.total_requests;
        });

        chartData.push(monthData);
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
          icon={<IconAlertTriangle size={16} />}
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
            const totalUsed = Object.values(latestData).reduce(
              (sum: number, usage) => sum + usage.characters_used,
              0
            );
            const totalQuota = Object.values(latestData).reduce(
              (sum: number, usage) => sum + usage.quota,
              0
            );

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
        </SimpleGrid>

        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Tabs defaultValue='overview' keepMounted={false}>
              <Tabs.List>
                <Tabs.Tab
                  value='overview'
                  leftSection={<IconTrendingUp size={16} />}
                >
                  Overview
                </Tabs.Tab>
                <Tabs.Tab
                  value='monthly'
                  leftSection={<IconTrendingUp size={16} />}
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

                  {serviceUsage && (
                    <Card shadow='sm' padding='md' radius='md' withBorder>
                      <Text size='md' fw={500} mb='md'>
                        {serviceUsage.service.toUpperCase()} Translation Service
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
                          {serviceUsage.data.map((item, index) => {
                            const percentage = getQuotaPercentage(
                              item.characters_used || 0,
                              item.quota || 0
                            );
                            return (
                              <Table.Tr
                                key={`${item.month}-${item.usage_type}-${index}`}
                              >
                                <Table.Td>{formatMonth(item.month)}</Table.Td>
                                <Table.Td>
                                  <Badge variant='outline' size='sm'>
                                    {item.usage_type}
                                  </Badge>
                                </Table.Td>
                                <Table.Td>
                                  {formatNumber(item.characters_used || 0)}
                                </Table.Td>
                                <Table.Td>
                                  {formatNumber(item.requests_made || 0)}
                                </Table.Td>
                                <Table.Td>
                                  {formatNumber(item.quota || 0)}
                                </Table.Td>
                                <Table.Td>
                                  <Group gap='xs'>
                                    <Text
                                      size='sm'
                                      c={getQuotaColor(percentage)}
                                    >
                                      {percentage.toFixed(1)}%
                                    </Text>
                                    <Progress
                                      value={percentage}
                                      color={getQuotaColor(percentage)}
                                      size='xs'
                                      w={60}
                                    />
                                  </Group>
                                </Table.Td>
                              </Table.Tr>
                            );
                          })}
                        </Table.Tbody>
                      </Table>
                    </Card>
                  )}

                  {!serviceUsage && (
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
              leftSection={<IconChartBar size={16} />}
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
                <YAxis />
                <Tooltip
                  formatter={(value: number, name: string) => [
                    formatNumber(value),
                    name.replace('_', ' ').toUpperCase(),
                  ]}
                />
                <Legend />
                {usageStats.services.map(service => (
                  <Line
                    key={`${service}_characters`}
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
                    type='monotone'
                    dataKey={`${service}_requests`}
                    stroke='#82ca9d'
                    name={`${service} Requests`}
                    strokeWidth={2}
                  />
                ))}
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
