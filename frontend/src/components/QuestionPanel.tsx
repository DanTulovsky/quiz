import React from 'react';
import { Box, Card, Center, Transition } from '@mantine/core';
import LoadingSpinner from './LoadingSpinner';

export interface QuestionPanelProps {
  headerRight?: React.ReactNode;
  children?: React.ReactNode;
  loading?: boolean;
  generating?: boolean;
  transitioning?: boolean;
  maxHeight?: string | number;
}

const QuestionPanel: React.FC<QuestionPanelProps> = ({
  headerRight,
  children,
  loading = false,
  generating = false,
  transitioning = false,
  maxHeight,
}) => {
  return (
    <Box>
      {headerRight}
      <Card
        shadow='sm'
        radius={10}
        withBorder
        p={0}
        style={{
          minHeight: '400px',
          ...(maxHeight ? { maxHeight } : {}),
          display: 'flex',
          flexDirection: 'column',
          overflow: 'visible',
        }}
      >
        {loading || generating || transitioning ? (
          <Center w='100%' h='100%'>
            <LoadingSpinner />
          </Center>
        ) : (
          <Transition
            mounted={!transitioning}
            transition='fade'
            duration={300}
            timingFunction='ease'
          >
            {styles => (
              <Box style={{ ...styles, width: '100%', height: '100%' }}>
                {children}
              </Box>
            )}
          </Transition>
        )}
      </Card>
    </Box>
  );
};

export default QuestionPanel;
