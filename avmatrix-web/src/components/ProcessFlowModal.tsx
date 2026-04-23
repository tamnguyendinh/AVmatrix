/**
 * Process Flow Modal
 *
 * Displays a Mermaid flowchart for a process in a centered modal popup.
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import { Copy, Focus, ZoomIn, ZoomOut } from 'lucide-react';
import mermaid from 'mermaid';
import DOMPurify from 'dompurify';
import { ProcessData, generateProcessMermaid } from '../lib/mermaid-generator';

interface ProcessFlowModalProps {
  process: ProcessData | null;
  onClose: () => void;
  onFocusInGraph?: (nodeIds: string[], processId: string) => void;
  isFullScreen?: boolean;
}

mermaid.initialize({
  startOnLoad: false,
  suppressErrorRendering: true, // Try to suppress if supported
  maxTextSize: 900000, // Increase from default 50000 to handle large combined diagrams
  theme: 'base',
  themeVariables: {
    primaryColor: '#e4dccf',
    primaryTextColor: '#1b1815',
    primaryBorderColor: '#2f2822',
    lineColor: '#6a5c4f',
    secondaryColor: '#eee7db',
    tertiaryColor: '#dcd2c3',
    mainBkg: '#e4dccf',
    nodeBorder: '#2f2822',
    clusterBkg: '#eee7db',
    clusterBorder: '#6a5c4f',
    titleColor: '#1b1815',
    edgeLabelBackground: '#eee7db',
  },
  flowchart: {
    curve: 'basis',
    padding: 50,
    nodeSpacing: 120,
    rankSpacing: 140,
    htmlLabels: true,
  },
});

// Suppress distinct syntax error overlay
mermaid.parseError = (err) => {
  // Suppress visual error - we handle errors in the render try/catch
  console.debug('Mermaid parse error (suppressed):', err);
};

export const ProcessFlowModal = ({
  process,
  onClose,
  onFocusInGraph,
  isFullScreen = false,
}: ProcessFlowModalProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const diagramRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Full process map gets higher default zoom (667%) and max zoom (3000%)
  const defaultZoom = isFullScreen ? 6.67 : 1;
  const maxZoom = isFullScreen ? 30 : 10;

  const [zoom, setZoom] = useState(defaultZoom);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });

  // Reset zoom when switching between full screen and regular mode
  useEffect(() => {
    setZoom(defaultZoom);
    setPan({ x: 0, y: 0 });
  }, [isFullScreen, defaultZoom]);

  // Handle zoom with scroll wheel
  useEffect(() => {
    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY * -0.001;
      setZoom((prev) => Math.min(Math.max(0.1, prev + delta), maxZoom));
    };

    const container = scrollContainerRef.current;
    if (container) {
      container.addEventListener('wheel', handleWheel, { passive: false });
      return () => container.removeEventListener('wheel', handleWheel);
    }
  }, [process, maxZoom]); // Re-attach when process or maxZoom changes

  // Handle keyboard zoom
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      if (e.key === '+' || e.key === '=') {
        setZoom((prev) => Math.min(prev + 0.2, maxZoom));
      } else if (e.key === '-' || e.key === '_') {
        setZoom((prev) => Math.max(prev - 0.2, 0.1));
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [maxZoom]);

  // Zoom in/out handlers
  const handleZoomIn = useCallback(() => {
    setZoom((prev) => Math.min(prev + 0.25, maxZoom));
  }, [maxZoom]);

  const handleZoomOut = useCallback(() => {
    setZoom((prev) => Math.max(prev - 0.25, 0.1));
  }, []);

  // Handle pan with mouse drag
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      setIsPanning(true);
      setPanStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
    },
    [pan],
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!isPanning) return;
      setPan({ x: e.clientX - panStart.x, y: e.clientY - panStart.y });
    },
    [isPanning, panStart],
  );

  const handleMouseUp = useCallback(() => {
    setIsPanning(false);
  }, []);

  const resetView = useCallback(() => {
    setZoom(defaultZoom);
    setPan({ x: 0, y: 0 });
  }, [defaultZoom]);

  // Render mermaid diagram
  useEffect(() => {
    if (!process || !diagramRef.current) return;

    const renderDiagram = async () => {
      try {
        // Check if we have raw mermaid code (from AI chat) or need to generate it
        const mermaidCode = process.rawMermaid
          ? process.rawMermaid
          : generateProcessMermaid(process);
        const id = `mermaid-${Date.now()}`;

        // Clear previous content
        diagramRef.current!.innerHTML = '';

        const { svg } = await mermaid.render(id, mermaidCode);
        if (!diagramRef.current) return;
        diagramRef.current!.innerHTML = DOMPurify.sanitize(svg, {
          USE_PROFILES: { svg: true, svgFilters: true },
          ADD_TAGS: ['foreignObject'],
        });
      } catch (error) {
        console.error('Mermaid render error:', error);
        const errorMessage = error instanceof Error ? error.message : String(error);
        const isSizeError = errorMessage.includes('Maximum') || errorMessage.includes('exceeded');

        diagramRef.current!.innerHTML = `
          <div class="text-center p-8">
            <div class="text-red-400 text-sm font-medium mb-2">
              ${isSizeError ? '📊 Diagram Too Large' : '⚠️ Render Error'}
            </div>
            <div class="text-slate-400 text-xs max-w-md">
              ${
                isSizeError
                  ? `This diagram has ${process.steps?.length || 0} steps and is too complex to render. Try viewing individual processes instead of "All Processes".`
                  : `Unable to render diagram. Steps: ${process.steps?.length || 0}`
              }
            </div>
          </div>
        `;
      }
    };

    renderDiagram();
  }, [process]);

  // Close on escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  // Close on backdrop click
  const handleBackdropClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === containerRef.current) {
        onClose();
      }
    },
    [onClose],
  );

  // Copy mermaid code to clipboard
  const handleCopyMermaid = useCallback(async () => {
    if (!process) return;
    const mermaidCode = generateProcessMermaid(process);
    await navigator.clipboard.writeText(mermaidCode);
  }, [process]);

  // Focus in graph
  const handleFocusInGraph = useCallback(() => {
    if (!process || !onFocusInGraph) return;
    const nodeIds = process.steps.map((s) => s.id);
    onFocusInGraph(nodeIds, process.id);
    onClose();
  }, [process, onFocusInGraph, onClose]);

  if (!process) return null;

  return (
    <div
      ref={containerRef}
      className="fixed inset-0 z-50 flex animate-fade-in items-center justify-center bg-overlay/60"
      onClick={handleBackdropClick}
      data-testid="process-modal"
    >
      <div
        className={`press-panel animate-scale-in relative flex flex-col overflow-hidden shadow-[var(--shadow-dropdown)] ${
          isFullScreen ? 'h-[95vh] w-[98%] max-w-none' : 'max-h-[90vh] w-[95%] max-w-5xl'
        }`}
      >
        <div className="relative z-10 border-b-[3px] border-border-default bg-base px-6 py-5">
          <p className="press-eyebrow">Process flow</p>
          <h2 className="press-title text-2xl">Process: {process.label}</h2>
        </div>

        <div
          ref={scrollContainerRef}
          className={`relative z-10 flex flex-1 items-center justify-center overflow-hidden bg-base p-8 ${isFullScreen ? 'min-h-[70vh]' : 'min-h-[400px]'}`}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          style={{ cursor: isPanning ? 'grabbing' : 'grab' }}
        >
          <div
            ref={diagramRef}
            className="h-fit w-fit origin-center transition-transform [&_.edgePath_.path]:stroke-border-default [&_.edgePath_.path]:stroke-2 [&_.marker]:fill-border-default"
            style={{
              transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            }}
          />
        </div>

        <div className="relative z-10 flex items-center justify-center gap-3 border-t-[3px] border-border-default bg-surface px-6 py-4">
          <div className="flex items-center gap-1 rounded-lg border-[2px] border-border-default bg-base p-1">
            <button
              onClick={handleZoomOut}
              className="press-ghost-button rounded-md p-2 text-text-secondary"
              title="Zoom out (-)"
            >
              <ZoomOut className="h-4 w-4" />
            </button>
            <span className="min-w-[3rem] px-2 text-center font-mono text-xs text-text-secondary">
              {Math.round(zoom * 100)}%
            </span>
            <button
              onClick={handleZoomIn}
              className="press-ghost-button rounded-md p-2 text-text-secondary"
              title="Zoom in (+)"
            >
              <ZoomIn className="h-4 w-4" />
            </button>
          </div>
          <button
            onClick={resetView}
            className="press-outline-button flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium"
            title="Reset zoom and pan"
          >
            Reset View
          </button>
          {onFocusInGraph && (
            <button
              onClick={handleFocusInGraph}
              className="press-filled-button flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-medium"
            >
              <Focus className="h-4 w-4" />
              Toggle Focus
            </button>
          )}
          <button
            onClick={handleCopyMermaid}
            className="press-outline-button flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-medium"
          >
            <Copy className="h-4 w-4" />
            Copy Mermaid
          </button>
          <button
            onClick={onClose}
            className="press-ghost-button rounded-lg px-5 py-2.5 text-sm font-medium"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
