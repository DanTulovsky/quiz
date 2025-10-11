import React, { useState } from 'react';
import {
  Modal,
  Group,
  Button,
  ScrollArea,
  Loader,
  Center,
  Text,
  Paper,
} from '@mantine/core';
import ReactDiffViewer from 'react-diff-viewer-continued';
import logger from '../utils/logger';

interface AIFixModalProps {
  opened: boolean;
  original: Record<string, unknown> | null;
  suggestion: Record<string, unknown> | null;
  loading?: boolean;
  onClose: () => void;
  onApply: () => Promise<void> | void;
}

// Format object with alphabetical ordering of keys (including content keys)
function formatQuestionForDiff(obj: Record<string, unknown> | null): string {
  if (!obj) return JSON.stringify({}, null, 2);
  const out: Record<string, unknown> = {};

  const topKeys = Object.keys(obj).sort();
  for (const k of topKeys) {
    // exclude fields that shouldn't appear in the diff
    if (k === 'change_reason') continue;
    if (k === 'content') {
      const content = obj['content'] as Record<string, unknown> | undefined;
      if (!content) continue;
      const cOut: Record<string, unknown> = {};
      const contentKeys = Object.keys(content).sort();
      for (const ck of contentKeys) {
        cOut[ck] = content[ck];
      }
      out['content'] = cOut;
    } else {
      out[k] = obj[k as keyof typeof obj];
    }
  }
  return JSON.stringify(out, null, 2);
}

const AIFixModal: React.FC<AIFixModalProps> = ({
  opened,
  original,
  suggestion,
  loading,
  onClose,
  onApply,
}) => {
  const left = formatQuestionForDiff(original || {});
  // suggestion sometimes contains merged top-level fields; show full suggestion with top-level fields
  const right = formatQuestionForDiff(
    (suggestion as Record<string, unknown>) || {}
  );
  const changeReason = (() => {
    if (!suggestion) return '';
    const v = suggestion['change_reason'];
    return typeof v === 'string' ? v : '';
  })();
  // Determine if there's additional context included in suggestion metadata
  const additionalContext = (() => {
    if (!suggestion) return '';
    const v = suggestion['additional_context'];
    return typeof v === 'string' ? v : '';
  })();
  const [applying, setApplying] = useState(false);

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span>AI Suggestion</span>
          {applying ? <Loader size='xs' /> : null}
        </div>
      }
      size='85vw'
      styles={{
        content: {
          width: '85vw',
          maxWidth: '95vw',
          minWidth: '480px',
          height: '92vh',
          padding: '8px 12px',
          boxSizing: 'border-box',
          overflow: 'auto',
          resize: 'both',
        },
      }}
    >
      {loading ? (
        <Center style={{ height: '60vh', flexDirection: 'column' }}>
          <Loader size='lg' />
          <Text size='sm' color='dimmed' mt='md'>
            Generating suggestion from AI. This may take a few seconds...
          </Text>
        </Center>
      ) : (
        <>
          {/* Top area: Reason spans full width. Buttons sit under the reason (right-aligned). */}
          <div style={{ margin: '0 12px 12px 12px' }}>
            {changeReason ? (
              <Paper
                withBorder
                p='sm'
                radius='sm'
                style={{
                  width: '100%',
                  background: 'var(--mantine-color-gray-0)',
                  whiteSpace: 'normal',
                  wordBreak: 'break-word',
                }}
              >
                <Text size='sm' fw={700} style={{ marginBottom: 6 }}>
                  Reason
                </Text>
                <Text size='sm' color='dimmed' style={{ whiteSpace: 'normal' }}>
                  {changeReason}
                </Text>
              </Paper>
            ) : null}
            {additionalContext ? (
              <Paper
                withBorder
                p='sm'
                radius='sm'
                style={{
                  marginTop: 8,
                  background: 'var(--mantine-color-gray-0)',
                }}
              >
                <Text size='sm' fw={700} style={{ marginBottom: 6 }}>
                  Additional Context
                </Text>
                <Text size='sm' color='dimmed' style={{ whiteSpace: 'normal' }}>
                  {additionalContext}
                </Text>
              </Paper>
            ) : null}

            <Group style={{ justifyContent: 'flex-end', marginTop: 8, gap: 8 }}>
              <Button variant='subtle' onClick={onClose}>
                Cancel
              </Button>
              <Button
                color='green'
                onClick={async () => {
                  if (!suggestion || applying) return;
                  try {
                    setApplying(true);
                    await Promise.resolve(onApply());
                  } catch (errUnknown) {
                  } finally {
                    setApplying(false);
                  }
                }}
                disabled={!suggestion || applying}
                loading={applying}
              >
                Apply Suggestion
              </Button>
            </Group>
          </div>

          {/* Resizable diff container only - allows horizontal resize so users can expand the diff area. */}
          <div
            style={{
              margin: '0 12px',
              display: 'flex',
              justifyContent: 'center',
            }}
          >
            <div
              style={{
                overflow: 'hidden',
                width: '100%',
                minWidth: 0,
                maxWidth: '100%',
                height: '72vh',
                borderRadius: 4,
                border: '1px solid var(--mantine-color-gray-2)',
                boxSizing: 'border-box',
              }}
            >
              <ScrollArea style={{ height: '100%' }} offsetScrollbars={true}>
                <div
                  style={{
                    marginBottom: 12,
                    padding: 8,
                    boxSizing: 'border-box',
                  }}
                >
                  <div
                    style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
                  >
                    <ReactDiffViewer
                      oldValue={left}
                      newValue={right}
                      splitView={true}
                      showDiffOnly={false}
                    />
                  </div>
                </div>
              </ScrollArea>
            </div>
          </div>
        </>
      )}
    </Modal>
  );
};

export default AIFixModal;
