import React, { useEffect } from 'react';
import logger from '../utils/logger';
import { Box, Input, Select } from '@mantine/core';
import { useTimezoneSelect } from 'react-timezone-select';

interface TimezoneSelectorProps {
  value: string;
  onChange: (timezone: string) => void;
  placeholder?: string;
  className?: string;
  isDisabled?: boolean;
}

const TimezoneSelector: React.FC<TimezoneSelectorProps> = ({
  value,
  onChange,
  placeholder = 'Select timezone...',
  className = '',
  isDisabled = false,
}) => {
  // Use react-timezone-select's hook to get options
  const { options } = useTimezoneSelect({ labelStyle: 'original' });

  // Auto-detect timezone if no value is set
  useEffect(() => {
    if (!value) {
      try {
        const detectedTimezone =
          Intl.DateTimeFormat().resolvedOptions().timeZone;
        onChange(detectedTimezone);
      } catch (error) {
        onChange('UTC');
      }
    }
  }, [value, onChange]);

  return (
    <Input.Wrapper label='Timezone' style={{ width: '100%' }}>
      <Box data-testid='timezone-select'>
        <Select
          data={options.map(opt => ({ value: opt.value, label: opt.label }))}
          value={value}
          onChange={val => onChange(val || '')}
          placeholder={placeholder}
          className={className}
          disabled={isDisabled}
          searchable
          clearable
          data-testid='timezone-select-input'
        />
      </Box>
    </Input.Wrapper>
  );
};

export default TimezoneSelector;
