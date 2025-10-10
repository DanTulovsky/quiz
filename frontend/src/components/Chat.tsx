import React, { useState, useRef, useEffect, useCallback } from 'react';
import logger from '../utils/logger';
import { PaperPlaneIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';
import {
  Maximize2,
  Minimize2,
  Square,
  Trash2,
  ChevronDown,
  Save,
} from 'lucide-react';
import {
  AnswerResponse,
  Question,
  useGetV1SettingsAiProviders,
  usePostV1AiConversations,
  usePostV1AiConversationsConversationIdMessages,
} from '../api/api';
import LoadingSpinner from './LoadingSpinner';
import ConfirmationModal from './ConfirmationModal';
import SyntaxHighlighter from 'react-syntax-highlighter';
import remarkGfm from 'remark-gfm';
import ReactMarkdown from 'react-markdown';
import { useAuth } from '../hooks/useAuth';
import { useHotkeys } from 'react-hotkeys-hook';
import { useMutation } from '@tanstack/react-query';
import {
  Paper,
  Stack,
  Group,
  Text,
  Button,
  Textarea,
  Divider,
  Box,
  Alert,
  Modal,
  TextInput,
  Badge,
} from '@mantine/core';

interface ChatProps {
  question: Question;
  answerContext?: AnswerResponse;
  isMaximized: boolean;
  setIsMaximized: (v: boolean) => void;
  showSuggestions?: boolean;
  setShowSuggestions?: (v: boolean) => void;
  onInputFocus?: () => void;
  onInputBlur?: () => void;
  onRegisterActions?: (actions: {
    clear: () => void;
    toggleMaximize: () => void;
  }) => void;
}

interface Message {
  sender: 'user' | 'ai';
  text: string;
  isThinking?: boolean;
}

interface ChatMessage {
  role: string;
  content: string;
}

const suggestedPrompts = [
  'Explain the grammar for this question in English',
  'Explain the correct answer for this question in English',
  'Give me another example of this concept',
  'Translate this question, text and options to English',
  'What is the difficulty level of this question?',
  'What are common mistakes for this question type?',
  'Can you break down this question step by step?',
  'What grammar rules should I remember?',
  'Translate the explanation to this question to English',
];

// Helper function to get drag position class based on coordinates
// const getDragPositionClass = (x: number, y: number): string => {
//   const distance = Math.sqrt(x * x + y * y);
//   if (distance <= 50) return 'chat-modal-drag-50';
//   if (distance <= 100) return 'chat-modal-drag-100';
//   if (distance <= 150) return 'chat-modal-drag-150';
//   if (distance <= 200) return 'chat-modal-drag-200';
//   if (distance <= 250) return 'chat-modal-drag-250';
//   if (distance <= 300) return 'chat-modal-drag-300';
//   if (distance <= 350) return 'chat-modal-drag-350';
//   if (distance <= 400) return 'chat-modal-drag-400';
//   if (distance <= 450) return 'chat-modal-drag-450';
//   return 'chat-modal-drag-500';
// };

// Shared message bubble for consistent formatting
const MessageBubble: React.FC<{
  msg: Message;
  onSave?: (messageText: string, messageIndex: number) => void;
  messageIndex: number;
  isLoading?: boolean;
}> = ({ msg, onSave, messageIndex, isLoading }) => (
  <Box
    p='sm'
    mb='xs'
    data-sender={msg.sender}
    style={{
      borderRadius: 'var(--mantine-radius-sm)',
      backgroundColor:
        msg.sender === 'user'
          ? 'var(--mantine-primary-color-1)'
          : 'var(--mantine-color-body)',
      textAlign: msg.sender === 'user' ? 'right' : 'left',
      marginLeft: msg.sender === 'user' ? 'auto' : '0',
      marginRight: msg.sender === 'user' ? '0' : 'auto',
      boxShadow: msg.sender === 'ai' ? 'var(--mantine-shadow-xs)' : 'none',
      maxWidth: '90%',
      border: '1px solid var(--mantine-color-border)',
      '&:hover': {
        borderColor: 'var(--mantine-color-hover)',
      },
    }}
  >
    <Box size='sm' style={{ maxWidth: 'none' }}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          code({ className, children, ...props }: any) {
            const match = /language-(\w+)/.exec(className || '');
            return match ? (
              <SyntaxHighlighter
                className='syntax-highlighter-vsc-dark'
                language={match[1]}
                PreTag='div'
                {...props}
              >
                {String(children).replace(/\n$/, '')}
              </SyntaxHighlighter>
            ) : (
              <code className={className} {...props}>
                {children}
              </code>
            );
          },
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          p: ({ children }: any) => (
            <Box mb='md' component='p'>
              {children}
            </Box>
          ),
        }}
      >
        {msg.text}
      </ReactMarkdown>
      {msg.sender === 'ai' && onSave && (
        <Group justify='flex-end' mt='xs'>
          <Button
            variant='light'
            size='xs'
            leftSection={<Save size={14} />}
            onClick={() => onSave(msg.text, messageIndex)}
            disabled={isLoading}
          >
            Save
          </Button>
        </Group>
      )}
    </Box>
  </Box>
);

// Shared ChatPanel component for both compact and maximized views
interface ChatPanelProps {
  isMaximized: boolean;
  providerDisplayName: string;
  modelDisplayName: string;
  showSuggestions: boolean;
  setShowSuggestions: (v: boolean) => void;
  suggestedPrompts: string[];
  handleSend: (msg?: string) => void;
  isLoading: boolean;
  input: string;
  setInput: (v: string) => void;
  inputRef: React.RefObject<HTMLInputElement | HTMLTextAreaElement>;
  messages: Message[];
  MessageBubble: React.FC<{
    msg: Message;
    onSave?: (messageText: string, messageIndex: number) => void;
    messageIndex: number;
    isLoading?: boolean;
  }>;
  bottomRef: React.RefObject<HTMLDivElement>;
  error: string | null;
  isThinking: boolean;
  handleStop: () => void;
  handleClear: () => void;
  toggleMaximize: () => void;
  showClearConfirm: boolean;
  confirmClear: () => void;
  suggestionsRef: React.RefObject<HTMLDivElement>;
  setShowClearConfirm: (v: boolean) => void;
  selectedSuggestionIndex: number;
  setSelectedSuggestionIndex: (index: number) => void;
  onSaveMessage?: (messageText: string, messageIndex: number) => void;
  onSaveConversation?: () => void;
  currentConversationId?: string | null;
}

// Helper to get the last AI message index
function getLastAIMessageIndex(messages: Message[]): number | null {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].sender === 'ai') return i;
  }
  return null;
}

const ChatPanel: React.FC<
  ChatPanelProps & { onInputFocus?: () => void; onInputBlur?: () => void }
> = ({
  isMaximized,
  providerDisplayName,
  modelDisplayName,
  showSuggestions,
  setShowSuggestions,
  suggestedPrompts,
  handleSend,
  isLoading,
  input,
  setInput,
  inputRef,
  messages,
  MessageBubble,
  bottomRef,
  error,
  isThinking,
  handleStop,
  handleClear,
  toggleMaximize,
  showClearConfirm,
  confirmClear,
  suggestionsRef,
  setShowClearConfirm,
  selectedSuggestionIndex,
  setSelectedSuggestionIndex,
  onSaveMessage,
  onSaveConversation,
  currentConversationId,
  onInputFocus,
  onInputBlur,
}) => {
  // For blinking fix: always render the last AI message bubble, with spinner if needed
  const lastAIIndex = getLastAIMessageIndex(messages);
  const renderedMessages = messages.map((msg, index) => {
    if (msg.sender === 'ai' && index === lastAIIndex) {
      return (
        <Box key={index}>
          <MessageBubble
            msg={msg}
            onSave={onSaveMessage}
            messageIndex={index}
            isLoading={isLoading}
          />
          {(isLoading || isThinking) && (
            <Group justify='center' align='center' gap={4}>
              <LoadingSpinner />
              <Text size='xs' c='dimmed'>
                AI is thinking...
              </Text>
            </Group>
          )}
        </Box>
      );
    }
    return (
      <MessageBubble
        key={index}
        msg={msg}
        onSave={onSaveMessage}
        messageIndex={index}
        isLoading={isLoading}
      />
    );
  });

  return (
    <Box
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: isMaximized ? '100%' : '100%',
        minHeight: 0,
      }}
    >
      {/* Header (pinned/sticky in maximized) */}
      <Group
        justify='space-between'
        align='center'
        style={{
          flexShrink: 0,
          position: 'sticky',
          top: 0,
          zIndex: 2,
          background: 'var(--mantine-color-body)',
        }}
      >
        <Group align='center' gap='md'>
          <div>
            <Text fw={600} size='lg'>
              Ask AI
            </Text>
            <Text size='xs' c='dimmed'>
              {providerDisplayName} • {modelDisplayName}
            </Text>
          </div>
          <Box
            ref={suggestionsRef}
            style={{ position: 'relative', width: 200 }}
          >
            <Button
              variant='subtle'
              size='xs'
              onClick={() => {
                setShowSuggestions(!showSuggestions);
                setSelectedSuggestionIndex(-1);
              }}
              disabled={isLoading}
              rightSection={<ChevronDown size={16} />}
              style={{
                fontWeight: 600,
                fontSize: 16,
                minWidth: 200,
                display: 'flex',
                alignItems: 'center',
              }}
            >
              Quick suggestions...
              <Badge
                ml={8}
                size='xs'
                color='gray'
                variant='filled'
                radius='sm'
                style={{
                  minWidth: 20,
                  padding: '0 6px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 600,
                  fontSize: 14,
                }}
              >
                Q
              </Badge>
            </Button>
            {showSuggestions && (
              <Paper
                withBorder
                shadow='xs'
                mt={2}
                style={{
                  position: 'absolute',
                  zIndex: 10,
                  left: 0,
                  minWidth: '100%',
                  width: 'max-content',
                  maxWidth: 400,
                  textAlign: 'left',
                }}
              >
                <Stack gap={0}>
                  {suggestedPrompts.map((prompt, index) => {
                    const hotkeyNumber =
                      index === 9 ? '0' : (index + 1).toString();
                    const isSelected = index === selectedSuggestionIndex;
                    return (
                      <Button
                        key={prompt}
                        variant={isSelected ? 'light' : 'subtle'}
                        c={isSelected ? 'blue' : 'gray'}
                        size='xs'
                        fullWidth
                        onClick={() => {
                          handleSend(prompt);
                          setSelectedSuggestionIndex(-1);
                        }}
                        disabled={isLoading}
                        style={{
                          justifyContent: 'flex-start',
                          width: '100%',
                        }}
                        styles={{
                          label: {
                            justifyContent: 'flex-start',
                            textAlign: 'left',
                            width: '100%',
                          },
                        }}
                      >
                        <Group
                          gap='xs'
                          justify='flex-start'
                          style={{ width: '100%' }}
                        >
                          <Badge
                            size='xs'
                            color='gray'
                            variant={isSelected ? 'light' : 'outline'}
                            radius='sm'
                            mr={4}
                          >
                            {hotkeyNumber}
                          </Badge>
                          <Text size='xs' style={{ flex: 1 }}>
                            {prompt}
                          </Text>
                        </Group>
                      </Button>
                    );
                  })}
                </Stack>
              </Paper>
            )}
          </Box>
        </Group>
        <Group gap={2}>
          <Button
            variant='subtle'
            c='gray'
            size='xs'
            onClick={onSaveConversation}
            title='Save Conversation'
            disabled={!currentConversationId || messages.length === 0}
          >
            <Save size={16} />
            <Badge ml={4} size='xs' color='gray' variant='filled' radius='sm'>
              S
            </Badge>
          </Button>
          <Button
            variant='subtle'
            c='gray'
            size='xs'
            onClick={handleClear}
            title='Clear Chat'
          >
            <Trash2 size={16} />
            <Badge ml={4} size='xs' color='gray' variant='filled' radius='sm'>
              D
            </Badge>
          </Button>
          <Button
            variant='subtle'
            c='gray'
            size='xs'
            onClick={toggleMaximize}
            title={isMaximized ? 'Minimize' : 'Maximize'}
          >
            {isMaximized ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
            <Badge ml={4} size='xs' color='gray' variant='filled' radius='sm'>
              M
            </Badge>
          </Button>
        </Group>
      </Group>
      {/* Scrollable messages area (only this should scroll) */}
      <Box
        data-testid='chat-messages-container'
        style={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <Stack gap='xs' pr={4} style={{ flex: 1 }}>
          {renderedMessages}
          <div ref={bottomRef} />
          {error && (
            <Alert c='red' icon={<ExclamationTriangleIcon />}>
              {error}
            </Alert>
          )}
        </Stack>
      </Box>
      {/* Input area (pinned at bottom) */}
      <Divider my='xs' />
      <Group align='flex-end' gap='xs' wrap='nowrap' style={{ flexShrink: 0 }}>
        <Box style={{ flex: 1, position: 'relative' }}>
          {isMaximized ? (
            <TextInput
              id='ai-chat-input'
              className='ai-chat-input'
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => {
                // Handle arrow key navigation for quick suggestions
                if (showSuggestions && suggestedPrompts.length > 0) {
                  if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    const newIndex =
                      selectedSuggestionIndex >= suggestedPrompts.length - 1
                        ? 0
                        : selectedSuggestionIndex + 1;
                    setSelectedSuggestionIndex(newIndex);
                    return;
                  }
                  if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    const newIndex =
                      selectedSuggestionIndex <= 0
                        ? suggestedPrompts.length - 1
                        : selectedSuggestionIndex - 1;
                    setSelectedSuggestionIndex(newIndex);
                    return;
                  }
                }

                if (showSuggestions && e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  e.stopPropagation();
                  handleSend(
                    suggestedPrompts[selectedSuggestionIndex] || input
                  );
                  setShowSuggestions(false);
                  setSelectedSuggestionIndex(-1);
                  return;
                }
                // Handle Enter for sending chat message when suggestions are closed
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  e.stopPropagation();
                  handleSend();
                }
              }}
              placeholder='Ask a follow-up question...'
              disabled={isLoading}
              style={{ flex: 1 }}
              ref={inputRef as React.RefObject<HTMLInputElement>}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
            />
          ) : (
            <Textarea
              id='ai-chat-input'
              className='ai-chat-input'
              value={input}
              onChange={e => setInput(e.target.value)}
              placeholder='Ask a follow-up question...'
              disabled={isLoading}
              minRows={1}
              maxRows={4}
              autosize
              style={{ flex: 1 }}
              onKeyDown={e => {
                // Handle arrow key navigation for quick suggestions
                if (showSuggestions && suggestedPrompts.length > 0) {
                  if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    const newIndex =
                      selectedSuggestionIndex >= suggestedPrompts.length - 1
                        ? 0
                        : selectedSuggestionIndex + 1;
                    setSelectedSuggestionIndex(newIndex);
                    return;
                  }
                  if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    const newIndex =
                      selectedSuggestionIndex <= 0
                        ? suggestedPrompts.length - 1
                        : selectedSuggestionIndex - 1;
                    setSelectedSuggestionIndex(newIndex);
                    return;
                  }
                }

                if (showSuggestions && e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  e.stopPropagation();
                  handleSend(
                    suggestedPrompts[selectedSuggestionIndex] || input
                  );
                  setShowSuggestions(false);
                  setSelectedSuggestionIndex(-1);
                  return;
                }
                // Handle Enter for sending chat message when suggestions are closed
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  e.stopPropagation();
                  handleSend();
                }
              }}
              ref={inputRef as React.RefObject<HTMLTextAreaElement>}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
            />
          )}
          <Badge
            size='xs'
            color='gray'
            variant='filled'
            radius='sm'
            style={{
              position: 'absolute',
              top: '50%',
              right: 8,
              transform: 'translateY(-50%)',
              zIndex: 1,
              pointerEvents: 'none',
            }}
          >
            C
          </Badge>
        </Box>
        {isLoading ? (
          <Button c='red' variant='filled' size='md' onClick={handleStop}>
            <Square size={18} />
          </Button>
        ) : (
          <Button
            c='blue'
            variant='filled'
            size='md'
            onClick={() => handleSend()}
            disabled={isLoading}
            aria-label='Send'
          >
            <PaperPlaneIcon color='white' />
            <Badge ml={6} size='xs' color='gray' variant='filled' radius='sm'>
              ↵
            </Badge>
          </Button>
        )}
      </Group>
      {/* hint removed from inside panel; rendered below the chat pane in compact view */}
      <ConfirmationModal
        isOpen={showClearConfirm}
        onClose={() => setShowClearConfirm(false)}
        onConfirm={confirmClear}
        title='Clear Chat'
        message='Are you sure you want to clear the chat?'
        confirmText='Clear'
        cancelText='Cancel'
      />
    </Box>
  );
};

export const Chat: React.FC<ChatProps> = ({
  question,
  answerContext,
  isMaximized,
  setIsMaximized,
  showSuggestions: showSuggestionsProp,
  setShowSuggestions: setShowSuggestionsProp,
  onInputFocus,
  onInputBlur,
  onRegisterActions,
}) => {
  const { user } = useAuth();
  const { data: providersData } = useGetV1SettingsAiProviders();
  const providers = providersData?.providers;

  // API mutations for saving conversations
  const createConversationMutation = useMutation({
    mutationFn: usePostV1AiConversations().mutateAsync,
  });

  const addMessageMutation = useMutation({
    mutationFn: usePostV1AiConversationsConversationIdMessages().mutateAsync,
  });

  // Get current provider and model names
  const currentProvider = providers?.find(p => p.code === user?.ai_provider);
  const currentModel = currentProvider?.models?.find(
    m => m.code === user?.ai_model
  );
  const providerDisplayName =
    currentProvider?.name || user?.ai_provider || 'Not configured';
  const modelDisplayName =
    currentModel?.name || user?.ai_model || 'Not configured';

  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isThinking, setIsThinking] = useState(false);
  const [showSuggestionsInternal, setShowSuggestionsInternal] = useState(false);
  const [currentConversationId, setCurrentConversationId] = useState<
    string | null
  >(null);
  const [, setIsSaving] = useState(false);
  const showSuggestions =
    showSuggestionsProp !== undefined
      ? showSuggestionsProp
      : showSuggestionsInternal;
  const setShowSuggestions =
    setShowSuggestionsProp || setShowSuggestionsInternal;
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [chatContainerHeight, setChatContainerHeight] = useState<number>(0);
  const [selectedSuggestionIndex, setSelectedSuggestionIndex] =
    useState<number>(-1);
  const abortControllerRef = useRef<AbortController | null>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);

  // Hotkey handlers for quick suggestions
  // 'q' key to expand and focus quick suggestions
  useHotkeys(
    'q',
    e => {
      if (!isLoading) {
        e.preventDefault();
        setShowSuggestions(!showSuggestions);
        // Scroll into view similar to 'c' behavior so dropdown is visible, then focus the button
        setTimeout(() => {
          if (!isMaximized) {
            const inputElement =
              inputRef.current ||
              (document.getElementById('ai-chat-input') as
                | HTMLInputElement
                | HTMLTextAreaElement
                | null);
            inputElement?.scrollIntoView({
              behavior: 'smooth',
              block: 'center',
            });
          }
          const suggestionsButton =
            suggestionsRef.current?.querySelector('button');
          if (suggestionsButton) {
            (suggestionsButton as HTMLElement).focus();
          }
        }, 0);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  // Arrow keys for quick suggestions navigation (when dropdown is open)
  useHotkeys(
    'arrowup',
    e => {
      if (showSuggestions && suggestedPrompts.length > 0) {
        e.preventDefault();
        const newIndex =
          selectedSuggestionIndex <= 0
            ? suggestedPrompts.length - 1
            : selectedSuggestionIndex - 1;
        setSelectedSuggestionIndex(newIndex);
        return;
      }
      // Otherwise scroll chat window up (only when suggestions are closed)
      const messagesContainer = document.querySelector(
        '[data-testid="chat-messages-container"]'
      ) as HTMLElement;
      if (messagesContainer) {
        messagesContainer.scrollTop -= 100;
      }
    },
    { enableOnFormTags: true, preventDefault: true }
  );

  useHotkeys(
    'arrowdown',
    e => {
      if (showSuggestions && suggestedPrompts.length > 0) {
        e.preventDefault();
        const newIndex =
          selectedSuggestionIndex >= suggestedPrompts.length - 1
            ? 0
            : selectedSuggestionIndex + 1;
        setSelectedSuggestionIndex(newIndex);
        return;
      }
      // Otherwise scroll chat window down (only when suggestions are closed)
      const messagesContainer = document.querySelector(
        '[data-testid="chat-messages-container"]'
      ) as HTMLElement;
      if (messagesContainer) {
        messagesContainer.scrollTop += 100;
      }
    },
    { enableOnFormTags: true, preventDefault: true }
  );

  // Enter key for executing selected suggestion (only when suggestions are open)
  useHotkeys(
    'enter',
    e => {
      if (
        showSuggestions &&
        selectedSuggestionIndex >= 0 &&
        selectedSuggestionIndex < suggestedPrompts.length
      ) {
        e.preventDefault();
        handleSend(suggestedPrompts[selectedSuggestionIndex]);
        setShowSuggestions(false);
        setSelectedSuggestionIndex(-1);
        return;
      }
    },
    { enableOnFormTags: true, preventDefault: true }
  );

  // Number keys 0-9 for selecting quick suggestions (when dropdown is open)
  useHotkeys(
    ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'],
    e => {
      if (showSuggestions && !isLoading) {
        e.preventDefault();
        const key = e.key;
        let index: number;

        if (key === '0') {
          index = 9; // 0 key selects the 10th item (index 9)
        } else {
          index = parseInt(key) - 1; // 1-9 keys select items 0-8
        }

        if (index >= 0 && index < suggestedPrompts.length) {
          handleSend(suggestedPrompts[index]);
          setShowSuggestions(false);
          setSelectedSuggestionIndex(-1);
        }
      }
    },
    { enableOnFormTags: false, preventDefault: true, enabled: showSuggestions }
  );

  // Handle 'd' key for clearing chat
  useHotkeys(
    'd',
    e => {
      e.preventDefault();
      setShowClearConfirm(true);
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  // Handle 'm' key for maximizing/minimizing chat
  useHotkeys(
    'm',
    e => {
      e.preventDefault();
      toggleMaximize();
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  // Handle ESC key to close quick suggestions dropdown or reset focus
  useHotkeys(
    'escape',
    e => {
      if (showSuggestions) {
        e.preventDefault();
        setShowSuggestions(false);
        setSelectedSuggestionIndex(-1);
        return;
      }

      // If input is focused, blur it to reset focus state
      const activeElement = document.activeElement as HTMLElement | null;
      if (
        activeElement &&
        (activeElement.tagName === 'INPUT' ||
          activeElement.tagName === 'TEXTAREA' ||
          activeElement.isContentEditable)
      ) {
        e.preventDefault();
        activeElement.blur();
        // Call onInputBlur to update the focus state
        onInputBlur?.();
      }
    },
    { enableOnFormTags: true, preventDefault: true }
  );

  // Auto-focus input when maximized
  useEffect(() => {
    if (isMaximized) {
      // Longer delay to ensure the modal is fully rendered and the input is available
      const timer = setTimeout(() => {
        const inputElement = inputRef.current;
        if (inputElement) {
          inputElement.focus();
        } else {
          // Fallback: try to find the input by ID
          const inputById = document.getElementById('ai-chat-input') as
            | HTMLInputElement
            | HTMLTextAreaElement;
          if (inputById) {
            inputById.focus();
          }
        }
      }, 200);

      return () => clearTimeout(timer);
    }
  }, [isMaximized]);

  // Removed the effect that was causing issues with focus state
  // The focus/blur callbacks on the input elements will handle the state properly

  // Handle 'c' key for focusing chat input (only when not maximized)
  useHotkeys(
    'c',
    e => {
      if (inputRef.current && !isMaximized) {
        e.preventDefault();
        inputRef.current.focus();
        // The onFocus event will handle updating the keyboard shortcuts display
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  // Calculate available height for the chat messages container
  useEffect(() => {
    const calculateHeight = () => {
      if (!chatContainerRef.current || isMaximized) return;

      const chatContainer = chatContainerRef.current;
      const containerTop = chatContainer.getBoundingClientRect().top;
      const windowHeight = window.innerHeight;

      // Reserve more space for all UI elements and padding
      // Header (50px) + Suggestions (50px) + Input area (60px) + Padding/margins (60px)
      const reservedSpace = 220;
      const availableHeight = windowHeight - containerTop - reservedSpace;

      // Set minimum and maximum heights for better UX
      const minHeight = 400;
      const maxHeight = Math.min(1600, windowHeight * 0.7);
      const calculatedHeight = Math.max(
        minHeight,
        Math.min(maxHeight, availableHeight)
      );

      setChatContainerHeight(calculatedHeight);
    };

    // Calculate on mount and window resize
    calculateHeight();
    window.addEventListener('resize', calculateHeight);

    // Also recalculate when the container might have moved (after content changes)
    const timeoutId = setTimeout(calculateHeight, 100);

    return () => {
      window.removeEventListener('resize', calculateHeight);
      clearTimeout(timeoutId);
    };
  }, [isMaximized, messages]);

  // Auto-scroll to top of new AI response when messages change or when chat is actively loading/thinking
  useEffect(() => {
    // Only scroll if there are messages or if chat is actively loading/thinking
    if (messages.length === 0 && !isLoading && !isThinking) {
      return;
    }

    const scrollToTopOfNewResponse = () => {
      requestAnimationFrame(() => {
        // Find the last AI message element
        const aiMessages = document.querySelectorAll('[data-sender="ai"]');
        if (aiMessages.length > 0) {
          const lastAIMessage = aiMessages[
            aiMessages.length - 1
          ] as HTMLElement;
          if (lastAIMessage) {
            // Scroll the message to the top of the viewport
            lastAIMessage.scrollIntoView({ behavior: 'auto', block: 'start' });
          }
        }
      });
    };
    scrollToTopOfNewResponse();
    const timeoutId = setTimeout(scrollToTopOfNewResponse, 100);
    return () => clearTimeout(timeoutId);
  }, [messages, isLoading, isThinking]);

  // Function to scroll to top of new response (can be called during streaming)
  const scrollToTopOfNewResponse = useCallback(() => {
    requestAnimationFrame(() => {
      // Find the last AI message element
      const aiMessages = document.querySelectorAll('[data-sender="ai"]');
      if (aiMessages.length > 0) {
        const lastAIMessage = aiMessages[aiMessages.length - 1] as HTMLElement;
        if (lastAIMessage) {
          // Scroll the message to the top of the viewport
          lastAIMessage.scrollIntoView({ behavior: 'auto', block: 'start' });
        }
      }
    });
  }, []);

  // Close suggestions dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        suggestionsRef.current &&
        !suggestionsRef.current.contains(event.target as Node)
      ) {
        setShowSuggestions(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [setShowSuggestions]);

  // Removed automatic focus on loading - only focus when user interacts with chat

  const handleSend = async (messageText?: string) => {
    const textToSend = messageText || input;
    if (!textToSend.trim()) return;

    const userMessage: Message = { sender: 'user', text: textToSend };
    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);
    setError(null);
    setShowSuggestions(false); // Close suggestions dropdown when sending
    // Refocus input after sending
    setTimeout(() => {
      inputRef.current?.focus();
    }, 0);

    // Add an empty AI message that we'll stream into
    const aiMessageIndex = messages.length + 1; // +1 for the user message we just added
    const aiMessage: Message = { sender: 'ai', text: '' };
    setMessages(prev => [...prev, aiMessage]);

    // Create abort controller for this request
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    try {
      // Convert current messages to conversation history format
      const conversationHistory: ChatMessage[] = messages.map(msg => ({
        role: msg.sender === 'user' ? 'user' : 'assistant',
        content: msg.text,
      }));

      // Use fetch with streaming for Server-Sent Events
      const response = await fetch('/v1/quiz/chat/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        credentials: 'include', // Include cookies for authentication
        signal: abortController.signal, // Add abort signal
        body: JSON.stringify({
          user_message: textToSend,
          question: question,
          answer_context: answerContext,
          conversation_history: conversationHistory,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) {
        throw new Error('No response body reader available');
      }

      let streamedText = '';

      while (true) {
        const { done, value } = await reader.read();

        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const rawData = line.slice(6).trim();
            if (!rawData) {
              if (streamedText === '' && !isThinking) {
                setIsThinking(true);
              }
              continue;
            }

            try {
              const parsedData = JSON.parse(rawData);

              if (parsedData && isThinking) {
                setIsThinking(false);
              }

              streamedText += parsedData;
              // Update the AI message in real-time
              setMessages(prev => {
                const newMessages = [...prev];
                newMessages[aiMessageIndex] = {
                  sender: 'ai',
                  text: streamedText,
                };
                return newMessages;
              });

              // Scroll to bottom during streaming with requestAnimationFrame for smoother performance
              requestAnimationFrame(scrollToTopOfNewResponse);
            } catch (e) {
              logger.error(
                'Failed to parse streaming data chunk as JSON:',
                rawData,
                e
              );
            }
          } else if (line.startsWith('event: error')) {
            // Handle error events
            throw new Error('Streaming error occurred');
          }
        }
      }
    } catch (e: unknown) {
      // Check if this was an abort
      if (e instanceof Error && e.name === 'AbortError') {
        // Update the AI message to show it was cancelled
        setMessages(prev => {
          const newMessages = [...prev];
          newMessages[aiMessageIndex] = {
            sender: 'ai',
            text: 'Response cancelled by user.',
          };
          return newMessages;
        });
        return;
      }

      const errorMsg =
        e instanceof Error ? e.message : 'An unexpected error occurred.';
      setError(errorMsg);
      // Update the AI message with error
      setMessages(prev => {
        const newMessages = [...prev];
        newMessages[aiMessageIndex] = {
          sender: 'ai',
          text: `Sorry, I encountered an error: ${errorMsg}`,
        };
        return newMessages;
      });
    } finally {
      setIsLoading(false);
      setIsThinking(false); // Reset thinking state
      abortControllerRef.current = null; // Clear the abort controller

      // Focus the input field after AI response completes
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100); // Small delay to ensure UI updates are complete
    }
  };

  const handleStop = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
  };

  const toggleMaximize = React.useCallback(() => {
    setIsMaximized(!isMaximized);
  }, [isMaximized, setIsMaximized]);

  const handleClear = React.useCallback(() => {
    if (messages.length > 0) {
      setShowClearConfirm(true);
    }
  }, [messages.length]);

  // Expose actions to parent to avoid DOM querying from pages
  useEffect(() => {
    if (onRegisterActions) {
      onRegisterActions({
        clear: () => handleClear(),
        toggleMaximize: () => toggleMaximize(),
      });
    }
    // It is fine if callback identity changes; consumers set fresh actions
  }, [onRegisterActions, handleClear, toggleMaximize]);

  const confirmClear = () => {
    setMessages([]);
    handleStop(); // Also cancel any pending requests
    setShowClearConfirm(false); // Always close the modal after clearing
  };

  const handleSaveMessage = async (
    messageText: string,
    messageIndex: number
  ) => {
    if (!user?.id || !question?.id) return;

    setIsSaving(true);
    try {
      // Find the corresponding user message for context
      const userMessageIndex = messageIndex - 1;
      const userMessage =
        userMessageIndex >= 0 ? messages[userMessageIndex] : null;

      // Create or use existing conversation
      let conversationId = currentConversationId;
      if (!conversationId) {
        const conversation = await createConversationMutation.mutateAsync({
          data: {
            title: `AI Chat - ${question.question_text?.substring(0, 50)}...`,
          },
        });
        conversationId = conversation.id;
        setCurrentConversationId(conversationId);
      }

      // Save the message
      await addMessageMutation.mutateAsync({
        conversationId,
        data: {
          question_id: question.id,
          role: 'assistant',
          content: {
            text: messageText,
            question_context: {
              question_text: question.question_text,
              explanation: question.explanation,
              correct_answer: question.correct_answer,
              options: question.options,
            },
            user_question: userMessage?.text || '',
          },
        },
      });

      // Show success feedback
      // You could add a toast notification here
    } catch (error) {
      logger.error('Failed to save message:', error);
      // You could add error feedback here
    } finally {
      setIsSaving(false);
    }
  };

  const handleSaveConversation = async () => {
    if (!user?.id || messages.length === 0) return;

    setIsSaving(true);
    try {
      // Create a new conversation or use existing one
      let conversationId = currentConversationId;
      if (!conversationId) {
        const conversation = await createConversationMutation.mutateAsync({
          data: {
            title: `AI Chat - ${question.question_text?.substring(0, 50)}...`,
          },
        });
        conversationId = conversation.id;
        setCurrentConversationId(conversationId);
      }

      // Save all messages to the conversation
      for (let i = 0; i < messages.length; i++) {
        const msg = messages[i];
        await addMessageMutation.mutateAsync({
          conversationId,
          data: {
            question_id: question.id,
            role: msg.sender === 'user' ? 'user' : 'assistant',
            content: {
              text: msg.text,
              question_context:
                i === 0
                  ? {
                      question_text: question.question_text,
                      explanation: question.explanation,
                      correct_answer: question.correct_answer,
                      options: question.options,
                    }
                  : undefined,
            },
          },
        });
      }

      // Show success feedback
      // You could add a toast notification here
    } catch (error) {
      logger.error('Failed to save conversation:', error);
      // You could add error feedback here
    } finally {
      setIsSaving(false);
    }
  };

  // Render ChatPanel inside Modal for maximized, or Paper for compact
  if (isMaximized) {
    return (
      <Modal
        opened={isMaximized}
        onClose={toggleMaximize}
        size='90%'
        fullScreen
        withCloseButton={false}
        styles={{
          overlay: {
            backgroundColor: 'rgba(20, 20, 20, 0.7)',
          },
          inner: {
            padding: '1rem',
          },
          content: {
            height: '100dvh',
            maxHeight: '100dvh',
            minHeight: 0,
            maxWidth: '64rem',
            margin: 'auto',
            transform: 'none',
            border: '2px solid var(--mantine-color-blue-7)',
            borderRadius: 'var(--mantine-radius-lg)',
            boxShadow: 'var(--mantine-shadow-xl)',
            background: 'var(--mantine-color-body)',
          },
        }}
      >
        <ChatPanel
          isMaximized={isMaximized}
          providerDisplayName={providerDisplayName}
          modelDisplayName={modelDisplayName}
          showSuggestions={showSuggestions}
          setShowSuggestions={setShowSuggestions}
          suggestedPrompts={suggestedPrompts}
          handleSend={handleSend}
          isLoading={isLoading}
          input={input}
          setInput={setInput}
          inputRef={inputRef}
          messages={messages}
          MessageBubble={MessageBubble}
          bottomRef={bottomRef}
          error={error}
          isThinking={isThinking}
          handleStop={handleStop}
          handleClear={handleClear}
          toggleMaximize={toggleMaximize}
          showClearConfirm={showClearConfirm}
          confirmClear={confirmClear}
          suggestionsRef={suggestionsRef}
          setShowClearConfirm={setShowClearConfirm}
          selectedSuggestionIndex={selectedSuggestionIndex}
          setSelectedSuggestionIndex={setSelectedSuggestionIndex}
          onSaveMessage={handleSaveMessage}
          onSaveConversation={handleSaveConversation}
          currentConversationId={currentConversationId}
          onInputFocus={onInputFocus}
          onInputBlur={onInputBlur}
        />
      </Modal>
    );
  }

  // Compact view
  return (
    <>
      <Paper
        ref={chatContainerRef}
        shadow='sm'
        radius='md'
        p='md'
        w='100%'
        withBorder
        style={{
          height: chatContainerHeight || 360,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
        }}
      >
        <ChatPanel
          isMaximized={false}
          providerDisplayName={providerDisplayName}
          modelDisplayName={modelDisplayName}
          showSuggestions={showSuggestions}
          setShowSuggestions={setShowSuggestions}
          suggestedPrompts={suggestedPrompts}
          handleSend={handleSend}
          isLoading={isLoading}
          input={input}
          setInput={setInput}
          inputRef={inputRef}
          messages={messages}
          MessageBubble={MessageBubble}
          bottomRef={bottomRef}
          error={error}
          isThinking={isThinking}
          handleStop={handleStop}
          handleClear={handleClear}
          toggleMaximize={toggleMaximize}
          showClearConfirm={showClearConfirm}
          confirmClear={confirmClear}
          suggestionsRef={suggestionsRef}
          setShowClearConfirm={setShowClearConfirm}
          selectedSuggestionIndex={selectedSuggestionIndex}
          setSelectedSuggestionIndex={setSelectedSuggestionIndex}
          onSaveMessage={handleSaveMessage}
          onSaveConversation={handleSaveConversation}
          currentConversationId={currentConversationId}
          onInputFocus={onInputFocus}
          onInputBlur={onInputBlur}
        />
      </Paper>
      {/* centered shortcut badge below the AI chat pane */}
      <Group justify='center' style={{ marginTop: 8 }}>
        <Badge
          size='xs'
          color='gray'
          variant='filled'
          radius='sm'
          style={{ opacity: 0.85, marginRight: 1 }}
        >
          ↑
        </Badge>
        <Badge size='xs' color='gray' variant='filled' radius='sm'>
          T
        </Badge>
      </Group>
    </>
  );
};
