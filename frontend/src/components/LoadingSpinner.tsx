import React from 'react';
import { Loader } from '@mantine/core';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'md',
  className,
}) => {
  const sizeMap = {
    sm: 'xs',
    md: 'sm',
    lg: 'md',
  } as const;

  return (
    <Loader
      size={sizeMap[size]}
      className={className}
      data-testid='loading-spinner'
    />
  );
};

export default LoadingSpinner;
