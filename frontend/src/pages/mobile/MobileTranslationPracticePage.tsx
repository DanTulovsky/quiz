import React from 'react';
import { Stack } from '@mantine/core';
import TranslationPracticePage from '../TranslationPracticePage';

const MobileTranslationPracticePage: React.FC = () => {
  // For now, reuse the same page; Mantine is responsive.
  return (
    <Stack p="xs">
      <TranslationPracticePage />
    </Stack>
  );
};

export default MobileTranslationPracticePage;
