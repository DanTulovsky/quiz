import React from 'react';
import {
  Group,
  Button,
  Text,
  Pagination as MantinePagination,
} from '@mantine/core';
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-react';

interface PaginationControlsProps {
  pagination: {
    currentPage: number;
    totalPages: number;
    totalItems: number;
    hasNextPage: boolean;
    hasPreviousPage: boolean;
  };
  onPageChange: (page: number) => void;
  onNext: () => void;
  onPrevious: () => void;
  isLoading?: boolean;
  variant?: 'desktop' | 'mobile';
}

export const PaginationControls: React.FC<PaginationControlsProps> = ({
  pagination,
  onPageChange,
  onNext,
  onPrevious,
  isLoading = false,
  variant = 'desktop',
}) => {
  const { currentPage, totalPages, totalItems, hasNextPage, hasPreviousPage } =
    pagination;

  if (totalPages <= 1) {
    return null;
  }

  if (variant === 'mobile') {
    return (
      <Group justify='space-between' align='center' py='sm'>
        <Text size='xs' c='dimmed'>
          {totalItems} items • Page {currentPage} of {totalPages}
        </Text>
        <Group gap='xs'>
          <Button
            variant='light'
            size='xs'
            leftSection={<ChevronLeft size={14} />}
            onClick={onPrevious}
            disabled={!hasPreviousPage || isLoading}
          >
            Previous
          </Button>
          <Button
            variant='filled'
            size='xs'
            rightSection={<ChevronRight size={14} />}
            onClick={onNext}
            disabled={!hasNextPage || isLoading}
          >
            Next
          </Button>
        </Group>
      </Group>
    );
  }

  return (
    <Group justify='space-between' align='center' py='md'>
      <Text size='sm' c='dimmed'>
        Showing {totalItems} items • Page {currentPage} of {totalPages}
      </Text>

      <Group gap='xs'>
        <Button
          variant='light'
          size='sm'
          leftSection={<ChevronsLeft size={14} />}
          onClick={() => onPageChange(1)}
          disabled={currentPage === 1 || isLoading}
        >
          First
        </Button>

        <Button
          variant='light'
          size='sm'
          leftSection={<ChevronLeft size={14} />}
          onClick={onPrevious}
          disabled={!hasPreviousPage || isLoading}
        >
          Previous
        </Button>

        <MantinePagination
          value={currentPage}
          onChange={onPageChange}
          total={totalPages}
          size='sm'
          withEdges
          disabled={isLoading}
        />

        <Button
          variant='light'
          size='sm'
          rightSection={<ChevronRight size={14} />}
          onClick={onNext}
          disabled={!hasNextPage || isLoading}
        >
          Next
        </Button>

        <Button
          variant='light'
          size='sm'
          rightSection={<ChevronsRight size={14} />}
          onClick={() => onPageChange(totalPages)}
          disabled={currentPage === totalPages || isLoading}
        >
          Last
        </Button>
      </Group>
    </Group>
  );
};
