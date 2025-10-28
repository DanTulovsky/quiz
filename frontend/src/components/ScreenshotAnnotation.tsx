import React, { useEffect, useRef, useState } from 'react';
import {
  Modal,
  Stack,
  Button,
  Group,
  SegmentedControl,
  ColorSwatch,
  Tooltip,
  Text,
  ActionIcon,
} from '@mantine/core';
import {
  IconPencil,
  IconArrowRight,
  IconRectangle,
  IconLetterT,
  IconArrowBack,
  IconRotateClockwise,
  IconTrash,
  IconCheck,
  IconX,
} from '@tabler/icons-react';
import * as fabric from 'fabric';

interface ScreenshotAnnotationProps {
  screenshotData: string;
  onSave: (annotatedImageData: string) => void;
  onCancel: () => void;
}

type Tool = 'pencil' | 'arrow' | 'rectangle' | 'text';

const COLORS = [
  { value: '#FF0000', label: 'Red' },
  { value: '#FFFF00', label: 'Yellow' },
  { value: '#00FF00', label: 'Green' },
  { value: '#0000FF', label: 'Blue' },
];

const ScreenshotAnnotation: React.FC<ScreenshotAnnotationProps> = ({
  screenshotData,
  onSave,
  onCancel,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const fabricCanvasRef = useRef<fabric.Canvas | null>(null);
  const [tool, setTool] = useState<Tool>('pencil');
  const [color, setColor] = useState('#FF0000');
  // isDrawing state is used but linting doesn't detect the reference inside useEffect
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [isDrawing, setIsDrawing] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);

  // Save canvas state for undo/redo
  const saveState = () => {
    if (!fabricCanvasRef.current) return;

    const state = fabricCanvasRef.current.toJSON();
    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push(JSON.stringify(state));
    setHistory(newHistory);
    setHistoryIndex(newHistory.length - 1);
  };

  // Initialize fabric canvas
  useEffect(() => {
    if (!canvasRef.current) return;

    const canvas = new fabric.Canvas(canvasRef.current, {
      width: 800,
      height: 600,
      backgroundColor: '#ffffff',
    });

    fabricCanvasRef.current = canvas;

    // Load background image
    fabric.Image.fromURL(
      screenshotData,
      img => {
        const canvas = fabricCanvasRef.current;
        if (!canvas) return;

        // Scale image to fit canvas
        const scale = Math.min(
          canvas.width! / img.width!,
          canvas.height! / img.height!
        );
        img.set({
          left: 0,
          top: 0,
          scaleX: scale,
          scaleY: scale,
          selectable: false,
          evented: false,
        });

        canvas.setBackgroundImage(img, canvas.renderAll.bind(canvas));
        saveState();
      },
      { crossOrigin: 'anonymous' }
    );

    return () => {
      canvas.dispose();
    };
  }, [screenshotData]);

  // Handle tool-specific mouse events
  useEffect(() => {
    const canvas = fabricCanvasRef.current;
    if (!canvas) return;

    const handleMouseDown = (options: fabric.IEvent) => {
      const pointer = canvas.getPointer(options.e);
      const x = pointer.x;
      const y = pointer.y;

      if (tool === 'pencil') {
        setIsDrawing(true);
        canvas.isDrawingMode = true;
        canvas.freeDrawingBrush.width = 3;
        canvas.freeDrawingBrush.color = color;
      } else if (tool === 'arrow') {
        const line = new fabric.Line([x, y, x, y], {
          stroke: color,
          strokeWidth: 3,
          selectable: true,
        });
        canvas.add(line);

        const updateLine = (e: fabric.IEvent) => {
          const pointer = canvas.getPointer(e.e);
          line.set({ x2: pointer.x, y2: pointer.y });
          canvas.renderAll();
        };

        const finishLine = () => {
          canvas.off('mouse:move', updateLine);
          canvas.off('mouse:up', finishLine);
          addArrowHead(line);
          saveState();
        };

        canvas.on('mouse:move', updateLine);
        canvas.on('mouse:up', finishLine);
      } else if (tool === 'rectangle') {
        const rect = new fabric.Rect({
          left: x,
          top: y,
          width: 0,
          height: 0,
          stroke: color,
          strokeWidth: 3,
          fill: 'transparent',
          selectable: true,
        });
        canvas.add(rect);

        const updateRect = (e: fabric.IEvent) => {
          const pointer = canvas.getPointer(e.e);
          const width = pointer.x - x;
          const height = pointer.y - y;
          rect.set({ width: Math.abs(width), height: Math.abs(height) });
          if (width < 0) rect.set({ left: pointer.x });
          if (height < 0) rect.set({ top: pointer.y });
          canvas.renderAll();
        };

        const finishRect = () => {
          canvas.off('mouse:move', updateRect);
          canvas.off('mouse:up', finishRect);
          saveState();
        };

        canvas.on('mouse:move', updateRect);
        canvas.on('mouse:up', finishRect);
      } else if (tool === 'text') {
        const text = new fabric.IText('Double-click to edit', {
          left: x,
          top: y,
          fontFamily: 'Arial',
          fontSize: 16,
          fill: color,
          selectable: true,
        });
        canvas.add(text);
        canvas.setActiveObject(text);
        canvas.renderAll();
        saveState();
      }
    };

    const handleMouseUp = () => {
      if (tool === 'pencil') {
        setIsDrawing(false);
        canvas.isDrawingMode = false;
        saveState();
      }
    };

    canvas.on('mouse:down', handleMouseDown);
    canvas.on('mouse:up', handleMouseUp);

    return () => {
      canvas.off('mouse:down', handleMouseDown);
      canvas.off('mouse:up', handleMouseUp);
    };
  }, [tool, color, saveState]);

  // Add arrow head to line
  const addArrowHead = (line: fabric.Line) => {
    if (!fabricCanvasRef.current) return;

    const angle = Math.atan2(line.y2! - line.y1!, line.x2! - line.x1!);
    const arrowLength = 15;
    const arrowWidth = 10;

    const arrowHead = new fabric.Polygon(
      [
        { x: 0, y: 0 },
        { x: arrowLength, y: arrowWidth / 2 },
        { x: arrowLength, y: -arrowWidth / 2 },
      ],
      {
        left: line.x2!,
        top: line.y2!,
        angle: (angle * 180) / Math.PI,
        originX: 'center',
        originY: 'center',
        fill: color,
        stroke: color,
        strokeWidth: 1,
        selectable: true,
      }
    );

    fabricCanvasRef.current.add(arrowHead);
    fabricCanvasRef.current.renderAll();
  };

  // Undo last action
  const undo = () => {
    if (historyIndex > 0) {
      const newIndex = historyIndex - 1;
      setHistoryIndex(newIndex);
      if (fabricCanvasRef.current) {
        fabricCanvasRef.current.loadFromJSON(
          history[newIndex],
          fabricCanvasRef.current.renderAll.bind(fabricCanvasRef.current)
        );
      }
    }
  };

  // Redo last undone action
  const redo = () => {
    if (historyIndex < history.length - 1) {
      const newIndex = historyIndex + 1;
      setHistoryIndex(newIndex);
      if (fabricCanvasRef.current) {
        fabricCanvasRef.current.loadFromJSON(
          history[newIndex],
          fabricCanvasRef.current.renderAll.bind(fabricCanvasRef.current)
        );
      }
    }
  };

  // Clear all annotations
  const clearAll = () => {
    if (fabricCanvasRef.current) {
      const objects = fabricCanvasRef.current.getObjects();
      objects.forEach(obj => fabricCanvasRef.current!.remove(obj));
      fabricCanvasRef.current.renderAll();
      saveState();
    }
  };

  // Save annotated image
  const handleSave = () => {
    if (!fabricCanvasRef.current) return;

    const dataUrl = fabricCanvasRef.current.toDataURL({
      format: 'jpeg',
      quality: 0.7,
      multiplier: 1,
    });
    onSave(dataUrl);
  };

  const canUndo = historyIndex > 0;
  const canRedo = historyIndex < history.length - 1;

  return (
    <Modal
      opened={true}
      onClose={onCancel}
      title='Annotate Screenshot'
      fullScreen
      padding={0}
    >
      <Stack gap={0}>
        {/* Toolbar */}
        <Group
          p='md'
          style={{
            borderBottom: '1px solid var(--mantine-color-gray-3)',
            backgroundColor: 'var(--mantine-color-gray-0)',
          }}
        >
          {/* Tool Selection */}
          <SegmentedControl
            data={[
              {
                value: 'pencil',
                label: (
                  <Tooltip label='Freehand' withArrow>
                    <ActionIcon variant='subtle'>
                      <IconPencil size={18} />
                    </ActionIcon>
                  </Tooltip>
                ),
              },
              {
                value: 'arrow',
                label: (
                  <Tooltip label='Arrow' withArrow>
                    <ActionIcon variant='subtle'>
                      <IconArrowRight size={18} />
                    </ActionIcon>
                  </Tooltip>
                ),
              },
              {
                value: 'rectangle',
                label: (
                  <Tooltip label='Rectangle' withArrow>
                    <ActionIcon variant='subtle'>
                      <IconRectangle size={18} />
                    </ActionIcon>
                  </Tooltip>
                ),
              },
              {
                value: 'text',
                label: (
                  <Tooltip label='Text' withArrow>
                    <ActionIcon variant='subtle'>
                      <IconLetterT size={18} />
                    </ActionIcon>
                  </Tooltip>
                ),
              },
            ]}
            value={tool}
            onChange={value => setTool(value as Tool)}
          />

          {/* Color Swatches */}
          <Group gap='xs'>
            <Text size='sm' c='dimmed'>
              Color:
            </Text>
            {COLORS.map((c, index) => (
              <Tooltip key={index} label={c.label} withArrow>
                <ColorSwatch
                  color={c.value}
                  size={24}
                  style={{ cursor: 'pointer' }}
                  onClick={() => setColor(c.value)}
                />
              </Tooltip>
            ))}
          </Group>

          {/* Action Buttons */}
          <Group gap='xs' ml='auto'>
            <Tooltip label='Undo' withArrow>
              <ActionIcon
                variant='subtle'
                onClick={undo}
                disabled={!canUndo}
                aria-label='Undo'
              >
                <IconArrowBack size={18} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label='Redo' withArrow>
              <ActionIcon
                variant='subtle'
                onClick={redo}
                disabled={!canRedo}
                aria-label='Redo'
              >
                <IconRotateClockwise size={18} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label='Clear All' withArrow>
              <ActionIcon
                variant='subtle'
                color='red'
                onClick={clearAll}
                aria-label='Clear All'
              >
                <IconTrash size={18} />
              </ActionIcon>
            </Tooltip>
          </Group>

          {/* Save/Cancel Buttons */}
          <Group gap='xs'>
            <Button
              leftSection={<IconCheck />}
              onClick={handleSave}
              variant='filled'
            >
              Done
            </Button>
            <Button leftSection={<IconX />} onClick={onCancel} variant='subtle'>
              Cancel
            </Button>
          </Group>
        </Group>

        {/* Canvas */}
        <div style={{ padding: '16px', overflow: 'auto', flex: 1 }}>
          <canvas ref={canvasRef} />
        </div>
      </Stack>
    </Modal>
  );
};

export default ScreenshotAnnotation;
