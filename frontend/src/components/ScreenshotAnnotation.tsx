import React, { useEffect, useRef, useState } from 'react';
import {
  Modal,
  Stack,
  Button,
  Group as MantineGroup,
  SegmentedControl,
  ColorSwatch,
  Tooltip,
  Text,
  ActionIcon,
} from '@mantine/core';
import { IconCheck, IconX } from '@tabler/icons-react';
import * as TablerIcons from '@tabler/icons-react';
import {
  Canvas,
  FabricImage,
  Line,
  Rect,
  IText,
  Polygon,
  TEvent,
  PencilBrush,
  util,
  Group as FabricGroup,
  FabricObject,
} from 'fabric';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<React.SVGProps<SVGSVGElement>>
>;
const IconPencil = tablerIconMap.IconPencil || (() => null);
const IconArrowRight = tablerIconMap.IconArrowRight || (() => null);
const IconRectangle =
  tablerIconMap.IconRectangle || tablerIconMap.IconSquare || (() => null);
const IconLetterT = tablerIconMap.IconLetterT || (() => null);
const IconArrowBack =
  tablerIconMap.IconArrowBack || tablerIconMap.IconChevronLeft || (() => null);
const IconRotateClockwise =
  tablerIconMap.IconRotateClockwise ||
  tablerIconMap.IconRefresh ||
  (() => null);
const IconTrash = tablerIconMap.IconTrash || (() => null);

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
  console.log(
    'ScreenshotAnnotation component rendering, screenshotData length:',
    screenshotData?.length
  );
  const [canvasElement, setCanvasElement] = useState<HTMLCanvasElement | null>(
    null
  );
  const fabricCanvasRef = useRef<Canvas | null>(null);
  const textJustExitedEditingRef = useRef(false);
  const [tool, setTool] = useState<Tool>('pencil');
  const [color, setColor] = useState('#FF0000');
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);

  // Callback ref to ensure we track when canvas mounts
  const canvasCallbackRef = (node: HTMLCanvasElement | null) => {
    if (node && !canvasElement) {
      console.log('Canvas callback ref called with node:', node);
      setCanvasElement(node);
    }
  };

  // Save canvas state for undo/redo
  const saveState = React.useCallback(() => {
    if (!fabricCanvasRef.current) return;

    const state = fabricCanvasRef.current.toJSON();
    setHistory(prev => {
      const newHistory = prev.slice(0, historyIndex + 1);
      newHistory.push(JSON.stringify(state));
      setHistoryIndex(newHistory.length - 1);
      return newHistory;
    });
  }, [historyIndex]);

  // Initialize fabric canvas
  useEffect(() => {
    console.log('useEffect triggered, canvasElement:', canvasElement);
    if (!canvasElement) {
      console.log('Canvas element not ready, exiting early');
      return;
    }

    // Store the current canvas ref to check if it's still valid in cleanup
    let isMounted = true;

    // Clean up any existing canvas
    if (fabricCanvasRef.current) {
      fabricCanvasRef.current.dispose();
      fabricCanvasRef.current = null;
    }

    console.log('Initializing fabric canvas...');
    const canvas = new Canvas(canvasElement, {
      width: 1200,
      height: 800,
      renderOnAddRemove: true,
    });

    // Ensure both canvases (lower and upper) have proper styling
    // Fabric.js creates a wrapper div and two canvases
    setTimeout(() => {
      const wrapperDiv = canvasElement?.parentElement;
      if (wrapperDiv) {
        const canvases = wrapperDiv.querySelectorAll('canvas');
        console.log('Found canvases:', canvases.length);
        canvases.forEach((c, idx) => {
          console.log(`Canvas ${idx}:`, c.className, c.width, c.height);
          c.style.display = 'block';
          if (idx === 0) {
            // Lower canvas - has the content
            c.style.border = '1px solid #e0e0e0';
            c.style.backgroundColor = '#ffffff';
          }
        });
      }
    }, 100);

    // Only set the ref if still mounted
    if (isMounted) {
      fabricCanvasRef.current = canvas;
    }

    // Load image with direct HTMLImageElement to ensure proper loading
    const img = document.createElement('img');
    img.crossOrigin = 'anonymous';

    img.onload = () => {
      console.log('Image loaded, dimensions:', img.width, img.height);

      // Create fabric image from the loaded HTML image
      try {
        console.log('Creating FabricImage...');
        const fabricImg = new FabricImage(img);
        console.log('FabricImage created successfully');

        // Calculate proper scale - ensure we use natural dimensions
        const imgWidth = img.naturalWidth || img.width;
        const imgHeight = img.naturalHeight || img.height;
        const scale = Math.min(1200 / imgWidth, 800 / imgHeight);

        console.log('Natural size:', imgWidth, imgHeight, 'Scale:', scale);

        // Set fabric image properties
        fabricImg.set({
          scaleX: scale,
          scaleY: scale,
          left: 0,
          top: 0,
          selectable: false,
          evented: false,
        });

        console.log('Fabric image created, adding to canvas...');

        // Add to canvas as background (first object added = bottom layer)
        canvas.add(fabricImg);

        // Set white background on the canvas
        canvas.backgroundColor = '#ffffff';

        // Force render multiple times to ensure both canvases update
        canvas.renderAll();
        canvas.requestRenderAll();

        console.log('Canvas background color:', canvas.backgroundColor);
        console.log('Canvas width/height:', canvas.width, canvas.height);

        console.log(
          'Canvas rendered, total objects:',
          canvas.getObjects().length
        );

        // Verify object is on canvas
        const objects = canvas.getObjects();
        if (objects.length > 0) {
          const firstObj = objects[0];
          console.log('First object on canvas:', {
            type: firstObj.type,
            width: firstObj.width,
            height: firstObj.height,
            scaleX: firstObj.scaleX,
            scaleY: firstObj.scaleY,
            left: firstObj.left,
            top: firstObj.top,
            getScaledWidth: firstObj.getScaledWidth(),
            getScaledHeight: firstObj.getScaledHeight(),
          });
        }

        // Save initial state
        setTimeout(() => {
          const state = canvas.toJSON();
          setHistory([JSON.stringify(state)]);
          setHistoryIndex(0);

          // If the initial tool is pencil, set up the brush and enable drawing mode
          if (tool === 'pencil') {
            canvas.freeDrawingBrush = new PencilBrush(canvas);
            canvas.freeDrawingBrush.width = 3;
            canvas.freeDrawingBrush.color = color;
            canvas.isDrawingMode = true;
            console.log('Initial pencil tool setup with drawing mode enabled');
          }
        }, 100);
      } catch (error) {
        console.error('Error creating FabricImage:', error);
      }
    };

    img.onerror = err => {
      console.error('Image load error:', err);
    };

    console.log('Setting image source...');
    img.src = screenshotData;

    return () => {
      isMounted = false;
      console.log('Cleaning up canvas...');
      // Only cleanup if this effect is truly unmounting, not just React StrictMode remount
      if (fabricCanvasRef.current) {
        fabricCanvasRef.current.dispose();
        fabricCanvasRef.current = null;
      }
    };
  }, [canvasElement, screenshotData]);

  // Update drawing mode and brush settings when tool or color changes
  useEffect(() => {
    const canvas = fabricCanvasRef.current;
    if (!canvas) return;

    // Always re-enable selection and make objects selectable when switching tools
    canvas.selection = true;
    const allObjects = canvas.getObjects();
    allObjects.forEach((obj, index) => {
      // Re-enable all objects except the background (index 0)
      if (index > 0) {
        obj.selectable = true;
        obj.evented = true;
      }
    });

    // Force Fabric.js to recalculate event handling
    canvas.calcOffset();

    if (tool === 'pencil') {
      // Create a new PencilBrush instance if it doesn't exist
      if (!canvas.freeDrawingBrush) {
        canvas.freeDrawingBrush = new PencilBrush(canvas);
      }
      canvas.freeDrawingBrush.width = 3;
      canvas.freeDrawingBrush.color = color;
      // Enable drawing mode immediately so drawing works on first click
      canvas.isDrawingMode = true;
      console.log('Drawing mode enabled for pencil tool');
    } else {
      canvas.isDrawingMode = false;
      console.log('Drawing mode disabled');
    }

    canvas.renderAll();
  }, [tool, color]);

  // Handle tool-specific mouse events
  useEffect(() => {
    const canvas = fabricCanvasRef.current;
    if (!canvas) return;

    const handleMouseDown = (options: TEvent) => {
      // Find what object was clicked (null if empty space)
      const clickedObject = canvas.findTarget(options.e);

      // If pencil tool is active and we clicked on an object, temporarily disable drawing to select it
      if (tool === 'pencil' && clickedObject) {
        const objects = canvas.getObjects();
        const clickedIndex = objects.indexOf(clickedObject);
        if (clickedIndex > 0) {
          // Clicked on an annotation - disable drawing mode and select it
          canvas.isDrawingMode = false;
          canvas.setActiveObject(clickedObject);
          canvas.renderAll();
          return;
        }
      }

      // Check if there's already a selected object
      const activeObject = canvas.getActiveObject();

      if (activeObject) {
        // Something is already selected
        if (clickedObject && clickedObject !== activeObject) {
          // Clicked on a different object - let Fabric handle switching selection
          const objects = canvas.getObjects();
          const clickedIndex = objects.indexOf(clickedObject);
          if (clickedIndex > 0) {
            // It's an annotation (not background) - let Fabric handle it
            return;
          }
        } else if (!clickedObject || clickedObject === canvas.getObjects()[0]) {
          // Clicked on empty space or background - deselect and re-enable drawing if pencil
          canvas.discardActiveObject();
          if (tool === 'pencil') {
            canvas.isDrawingMode = true;
          }
          canvas.renderAll();
          return;
        }
        // If we clicked on the same active object, let Fabric handle it (for rotation/resize controls)
        return;
      }

      // No active object - check if we clicked on an object to select it
      if (clickedObject) {
        const objects = canvas.getObjects();
        const clickedIndex = objects.indexOf(clickedObject);
        if (clickedIndex > 0) {
          // Clicked on an annotation - let Fabric handle selection
          return;
        }
      }

      // For pencil tool, drawing mode should already be enabled
      if (tool === 'pencil') {
        // Just return - drawing will happen automatically via isDrawingMode
        return;
      }

      // Special handling for text tool: prevent creating new text when clicking
      // anywhere that's not empty space (including on existing text or other objects)
      if (tool === 'text') {
        // If text just exited editing mode, don't create new text
        if (textJustExitedEditingRef.current) {
          textJustExitedEditingRef.current = false;
          return;
        }
        // If we clicked on any existing object (including text), don't create new text
        if (clickedObject) {
          // Let Fabric handle the interaction (selection, editing, etc.)
          return;
        }
        // If there was an active object (text being edited), clicking empty space
        // should just deselect it, not create new text
        const activeObj = canvas.getActiveObject();
        if (activeObj && activeObj.type === 'i-text') {
          canvas.discardActiveObject();
          canvas.renderAll();
          return;
        }
      }

      // Now we can safely create new shapes (nothing selected, clicked on empty space)
      const pointer = canvas.getPointer(options.e);
      const x = pointer.x;
      const y = pointer.y;

      if (tool === 'arrow') {
        let line: Line;
        let arrowGroup: FabricGroup;

        const startDrawing = () => {
          // Disable selection while drawing
          canvas.selection = false;
          const allObjects = canvas.getObjects();
          allObjects.forEach((obj, index) => {
            // Disable all objects except background
            if (index > 0) {
              obj.selectable = false;
              obj.evented = false;
            }
          });

          line = new Line([x, y, x, y], {
            stroke: color,
            strokeWidth: 3,
            selectable: false,
            evented: false,
          });
          canvas.add(line);
        };

        const updateLine = (e: TEvent) => {
          const pointer = canvas.getPointer(e.e);
          line.set({ x2: pointer.x, y2: pointer.y });
          canvas.renderAll();
        };

        const finishLine = () => {
          canvas.off('mouse:move', updateLine);
          canvas.off('mouse:up', finishLine);

          // Remove the temporary line first
          canvas.remove(line);

          // Re-enable selection
          canvas.selection = true;
          const allObjects = canvas.getObjects();
          allObjects.forEach((obj, index) => {
            // Re-enable all objects except the background (index 0)
            if (index > 0) {
              obj.selectable = true;
              obj.evented = true;
            }
          });

          // Force Fabric.js to recalculate event handling
          canvas.calcOffset();
          canvas.renderAll();

          // Create the arrow as a group
          const x1 = line.x1 || 0;
          const y1 = line.y1 || 0;
          const x2 = line.x2 || 0;
          const y2 = line.y2 || 0;

          // Calculate angle and length
          const angle = Math.atan2(y2 - y1, x2 - x1);
          const length = Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2);

          // Create line for the group (relative to group origin)
          const groupLine = new Line([0, 0, length, 0], {
            stroke: color,
            strokeWidth: 3,
            selectable: false,
          });

          // Create arrowhead
          const arrowLength = 15;
          const arrowWidth = 10;
          const arrowHead = new Polygon(
            [
              { x: length, y: 0 },
              { x: length - arrowLength, y: arrowWidth / 2 },
              { x: length - arrowLength, y: -arrowWidth / 2 },
            ],
            {
              fill: color,
              stroke: color,
              strokeWidth: 1,
              selectable: false,
            }
          );

          // Group line and arrowhead together
          arrowGroup = new FabricGroup([groupLine, arrowHead], {
            left: x1,
            top: y1,
            angle: (angle * 180) / Math.PI,
            selectable: true,
          });

          canvas.add(arrowGroup);
          canvas.renderAll();

          // Save state
          saveState();
        };

        startDrawing();
        canvas.on('mouse:move', updateLine);
        canvas.on('mouse:up', finishLine);
      } else if (tool === 'rectangle') {
        const rect = new Rect({
          left: x,
          top: y,
          width: 0,
          height: 0,
          stroke: color,
          strokeWidth: 3,
          fill: 'transparent',
          selectable: true,
          evented: true,
        });
        canvas.add(rect);

        const updateRect = (e: TEvent) => {
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

          console.log('finishRect called, rect:', {
            width: rect.width,
            height: rect.height,
            selectable: rect.selectable,
            evented: rect.evented,
          });

          // Save state
          saveState();
        };

        canvas.on('mouse:move', updateRect);
        canvas.on('mouse:up', finishRect);
      } else if (tool === 'text') {
        const text = new IText('Type here', {
          left: x,
          top: y,
          fontFamily: 'Arial',
          fontSize: 16,
          fill: color,
          selectable: true,
          editable: true,
        });
        canvas.add(text);
        canvas.setActiveObject(text);
        // Enter editing mode immediately
        text.enterEditing();
        text.selectAll();
        canvas.renderAll();
        // Save state after text is entered
        text.on('editing:exited', () => {
          // Set flag to prevent creating new text on the same click
          textJustExitedEditingRef.current = true;
          // Reset flag after a short delay
          setTimeout(() => {
            textJustExitedEditingRef.current = false;
          }, 100);

          setTimeout(() => {
            if (fabricCanvasRef.current) {
              const state = fabricCanvasRef.current.toJSON();
              setHistory(prev => {
                const newHistory = prev.slice(0, historyIndex + 1);
                newHistory.push(JSON.stringify(state));
                setHistoryIndex(newHistory.length - 1);
                return newHistory;
              });
            }
          }, 0);
        });
      }
    };

    // Save state after freehand drawing
    const handlePathCreated = () => {
      // Disable drawing mode after path is created so objects can be selected again
      if (canvas && tool === 'pencil') {
        canvas.isDrawingMode = false;
      }
      setTimeout(() => {
        if (fabricCanvasRef.current) {
          const state = fabricCanvasRef.current.toJSON();
          setHistory(prev => {
            const newHistory = prev.slice(0, historyIndex + 1);
            newHistory.push(JSON.stringify(state));
            setHistoryIndex(newHistory.length - 1);
            return newHistory;
          });
        }
      }, 0);
    };

    canvas.on('mouse:down', handleMouseDown);
    canvas.on('path:created', handlePathCreated);

    return () => {
      canvas.off('mouse:down', handleMouseDown);
      canvas.off('path:created', handlePathCreated);
    };
  }, [tool, color, historyIndex]);

  // Handle keyboard events for deleting selected objects
  useEffect(() => {
    const canvas = fabricCanvasRef.current;
    if (!canvas) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Delete or Backspace key
      if (e.key === 'Delete' || e.key === 'Backspace') {
        const activeObject = canvas.getActiveObject();
        if (activeObject) {
          // Don't delete the background image (first object)
          const objects = canvas.getObjects();
          if (objects[0] !== activeObject) {
            canvas.remove(activeObject);
            canvas.discardActiveObject();
            canvas.renderAll();
            saveState();
            e.preventDefault();
          }
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [historyIndex]);

  // Undo last action (preserve background image)
  const undo = () => {
    if (historyIndex > 0 && fabricCanvasRef.current) {
      const newIndex = historyIndex - 1;

      // Parse the history state to restore
      const historyData = JSON.parse(history[newIndex]);

      // Remove all annotation objects except the background (index 0)
      const currentObjects = fabricCanvasRef.current.getObjects();
      currentObjects
        .slice(1)
        .forEach(obj => fabricCanvasRef.current!.remove(obj));

      // Re-add annotation objects from history (skip the first one which is the background)
      if (historyData.objects && historyData.objects.length > 1) {
        const annotationObjects = historyData.objects.slice(1);

        util.enlivenObjects(annotationObjects).then(enlivenedObjects => {
          enlivenedObjects.forEach(obj => {
            fabricCanvasRef.current!.add(obj as FabricObject);
          });
          fabricCanvasRef.current!.renderAll();
        });
      } else {
        // No annotations in history, just render with background
        fabricCanvasRef.current.renderAll();
      }

      setHistoryIndex(newIndex);
    }
  };

  // Redo last undone action (preserve background image)
  const redo = () => {
    if (historyIndex < history.length - 1 && fabricCanvasRef.current) {
      const newIndex = historyIndex + 1;

      // Parse the history state to restore
      const historyData = JSON.parse(history[newIndex]);

      // Remove all annotation objects except the background (index 0)
      const currentObjects = fabricCanvasRef.current.getObjects();
      currentObjects
        .slice(1)
        .forEach(obj => fabricCanvasRef.current!.remove(obj));

      // Re-add annotation objects from history (skip the first one which is the background)
      if (historyData.objects && historyData.objects.length > 1) {
        const annotationObjects = historyData.objects.slice(1);

        util.enlivenObjects(annotationObjects).then(enlivenedObjects => {
          enlivenedObjects.forEach(obj => {
            fabricCanvasRef.current!.add(obj as FabricObject);
          });
          fabricCanvasRef.current!.renderAll();
        });
      } else {
        // No annotations in history, just render with background
        fabricCanvasRef.current.renderAll();
      }

      setHistoryIndex(newIndex);
    }
  };

  // Clear all annotations (but keep the background image)
  const clearAll = () => {
    if (fabricCanvasRef.current) {
      const objects = fabricCanvasRef.current.getObjects();
      // Skip the first object (the screenshot image) and remove everything else
      objects.slice(1).forEach(obj => fabricCanvasRef.current!.remove(obj));
      fabricCanvasRef.current.renderAll();
      saveState();
    }
  };

  // Save annotated image
  const handleSave = () => {
    console.log('Save button clicked');
    if (!fabricCanvasRef.current) {
      console.error('No canvas available');
      return;
    }

    try {
      console.log('Exporting canvas...');
      fabricCanvasRef.current.renderAll(); // Ensure everything is rendered
      const dataUrl = fabricCanvasRef.current.toDataURL({
        format: 'jpeg',
        quality: 0.7,
        multiplier: 1,
      });
      console.log('Canvas exported, calling onSave');
      onSave(dataUrl);
    } catch (error) {
      console.error('Failed to export canvas:', error);
      // Fallback: try without options
      try {
        const dataUrl = fabricCanvasRef.current.toDataURL();
        onSave(dataUrl);
      } catch (fallbackError) {
        console.error('Fallback export also failed:', fallbackError);
      }
    }
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
      styles={{
        body: {
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        },
      }}
    >
      <Stack gap={0} style={{ height: '100%' }} data-no-translate='true'>
        {/* Toolbar */}
        <MantineGroup
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
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '4px',
                      }}
                    >
                      <IconPencil style={{ width: 18, height: 18 }} />
                    </div>
                  </Tooltip>
                ),
              },
              {
                value: 'arrow',
                label: (
                  <Tooltip label='Arrow' withArrow>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '4px',
                      }}
                    >
                      <IconArrowRight style={{ width: 18, height: 18 }} />
                    </div>
                  </Tooltip>
                ),
              },
              {
                value: 'rectangle',
                label: (
                  <Tooltip label='Rectangle' withArrow>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '4px',
                      }}
                    >
                      <IconRectangle style={{ width: 18, height: 18 }} />
                    </div>
                  </Tooltip>
                ),
              },
              {
                value: 'text',
                label: (
                  <Tooltip label='Text' withArrow>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '4px',
                      }}
                    >
                      <IconLetterT style={{ width: 18, height: 18 }} />
                    </div>
                  </Tooltip>
                ),
              },
            ]}
            value={tool}
            onChange={value => value && setTool(value as Tool)}
          />

          {/* Color Swatches */}
          <MantineGroup gap='xs'>
            <Text size='sm' c='dimmed'>
              Color:
            </Text>
            {COLORS.map((c, index) => (
              <Tooltip key={index} label={c.label} withArrow>
                <ColorSwatch
                  color={c.value}
                  size={24}
                  style={{
                    cursor: 'pointer',
                    border:
                      color === c.value ? '3px solid #000' : '2px solid #ccc',
                    boxShadow:
                      color === c.value
                        ? '0 0 0 2px #fff, 0 0 0 4px #000'
                        : 'none',
                  }}
                  onClick={() => setColor(c.value)}
                />
              </Tooltip>
            ))}
          </MantineGroup>

          {/* Action Buttons */}
          <MantineGroup gap='xs' ml='auto'>
            <Tooltip label='Undo' withArrow>
              <ActionIcon
                variant='subtle'
                onClick={undo}
                disabled={!canUndo}
                aria-label='Undo'
              >
                <IconArrowBack style={{ width: 18, height: 18 }} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label='Redo' withArrow>
              <ActionIcon
                variant='subtle'
                onClick={redo}
                disabled={!canRedo}
                aria-label='Redo'
              >
                <IconRotateClockwise style={{ width: 18, height: 18 }} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label='Clear All' withArrow>
              <ActionIcon
                variant='subtle'
                color='red'
                onClick={clearAll}
                aria-label='Clear All'
              >
                <IconTrash style={{ width: 18, height: 18 }} />
              </ActionIcon>
            </Tooltip>
          </MantineGroup>

          {/* Save/Cancel Buttons */}
          <MantineGroup gap='xs'>
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
          </MantineGroup>
        </MantineGroup>

        {/* Canvas */}
        <div
          style={{
            padding: '16px',
            overflow: 'auto',
            flex: 1,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'flex-start',
            backgroundColor: '#f5f5f5',
          }}
        >
          <div
            style={{
              position: 'relative',
              width: 1200,
              height: 800,
            }}
            className='fabric-canvas-wrapper'
          >
            <canvas ref={canvasCallbackRef} width={1200} height={800} />
          </div>
        </div>
      </Stack>
    </Modal>
  );
};

export default ScreenshotAnnotation;
