import React, { useState, useRef, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { MermaidDiagram } from './MermaidDiagram';
import { ToolCallCard } from './ToolCallCard';
import { Copy, Check } from '@/lib/lucide-icons';

// Custom syntax theme
const customTheme = {
  ...vscDarkPlus,
  'pre[class*="language-"]': {
    ...vscDarkPlus['pre[class*="language-"]'],
    background: '#0a0a10',
    margin: 0,
    padding: '16px 0',
    fontSize: '13px',
    lineHeight: '1.6',
  },
  'code[class*="language-"]': {
    ...vscDarkPlus['code[class*="language-"]'],
    background: 'transparent',
    fontFamily: '"IBM Plex Mono", "Fira Code", monospace',
  },
};

interface MarkdownRendererProps {
  content: string;
  onLinkClick?: (href: string) => void;
  toolCalls?: any[]; // Keep flexible for now
  showCopyButton?: boolean;
}

export const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({
  content,
  onLinkClick,
  toolCalls,
  showCopyButton = false,
}) => {
  const [copied, setCopied] = useState(false);
  const copyTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      if (copyTimerRef.current) {
        clearTimeout(copyTimerRef.current);
      }
    };
  }, []);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      if (copyTimerRef.current) {
        clearTimeout(copyTimerRef.current);
      }
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  // Helper to format text for display (convert [[links]] to markdown links)
  const formatMarkdownForDisplay = (md: string) => {
    // Avoid rewriting inside fenced code blocks.
    const parts = md.split('```');
    for (let i = 0; i < parts.length; i += 2) {
      // Pattern 1: File grounding - [[file.ext]]
      parts[i] = parts[i].replace(
        /\[\[([a-zA-Z0-9_\-./\\]+\.[a-zA-Z0-9]+(?::\d+(?:[-–]\d+)?)?)\]\]/g,
        (_m, inner: string) => {
          const trimmed = inner.trim();
          const href = `code-ref:${encodeURIComponent(trimmed)}`;
          return `[${trimmed}](${href})`;
        },
      );

      // Pattern 2: Node grounding - [[Type:Name]]
      parts[i] = parts[i].replace(
        /\[\[(?:graph:)?(Class|Function|Method|Interface|File|Folder|Variable|Enum|Type|CodeElement):([^\]]+)\]\]/g,
        (_m, nodeType: string, nodeName: string) => {
          const trimmed = `${nodeType}:${nodeName.trim()}`;
          const href = `node-ref:${encodeURIComponent(trimmed)}`;
          return `[${trimmed}](${href})`;
        },
      );
    }
    return parts.join('```');
  };

  const handleLinkClick = React.useCallback(
    (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
      if (href.startsWith('code-ref:') || href.startsWith('node-ref:')) {
        e.preventDefault();
        onLinkClick?.(href);
      }
      // External links open in new tab (default behavior)
    },
    [onLinkClick],
  );

  const formattedContent = React.useMemo(() => formatMarkdownForDisplay(content), [content]);

  const markdownComponents = React.useMemo(
    () => ({
      a: ({ href, children, ...props }: any) => {
        const hrefStr = href || '';

        // Grounding links (Code refs & Node refs)
        if (hrefStr.startsWith('code-ref:') || hrefStr.startsWith('node-ref:')) {
          const isNodeRef = hrefStr.startsWith('node-ref:');
          const inner = decodeURIComponent(hrefStr.slice(isNodeRef ? 9 : 9)); // length is same? wait.. code-ref: (9), node-ref: (9). Yes.

          // Styles
          const baseParams =
            'code-ref-btn inline-flex items-center px-2 py-0.5 rounded-md font-mono text-[12px] !no-underline hover:!no-underline transition-colors';
          const colorParams = isNodeRef
            ? 'border-2 border-border-strong bg-base !text-text-primary visited:!text-text-primary hover:bg-surface'
            : 'border-2 border-border-default bg-surface !text-text-primary visited:!text-text-primary hover:border-border-strong hover:bg-base';

          return (
            <a
              href={hrefStr}
              onClick={(e) => handleLinkClick(e, hrefStr)}
              className={`${baseParams} ${colorParams}`}
              title={isNodeRef ? `View ${inner} in Code panel` : `Open in Code panel • ${inner}`}
              {...props}
            >
              <span className="text-inherit">{children}</span>
            </a>
          );
        }

        // External links
        return (
          <a
            href={hrefStr}
            className="text-border-strong underline underline-offset-2 hover:text-accent-dim"
            target="_blank"
            rel="noopener noreferrer"
            {...props}
          >
            {children}
          </a>
        );
      },
      code: ({ className, children, ...props }: any) => {
        const match = /language-(\w+)/.exec(className || '');
        const isInline = !className && !match;
        const codeContent = String(children).replace(/\n$/, '');

        if (isInline) {
          return <code {...props}>{children}</code>;
        }

        const language = match ? match[1] : 'text';

        // Render Mermaid diagrams
        if (language === 'mermaid') {
          return <MermaidDiagram code={codeContent} />;
        }

        return (
          <SyntaxHighlighter
            style={customTheme}
            language={language}
            PreTag="div"
            customStyle={{
              margin: 0,
              padding: '14px 16px',
              borderRadius: '12px',
              fontSize: '13px',
              background: '#0a0a10',
              border: '2px solid #7b634a',
            }}
          >
            {codeContent}
          </SyntaxHighlighter>
        );
      },
      pre: ({ children }: any) => <>{children}</>,
    }),
    [handleLinkClick],
  );

  return (
    <div className="text-sm text-text-primary">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        urlTransform={(url) => {
          if (url.startsWith('code-ref:') || url.startsWith('node-ref:')) return url;
          // Default behavior for http/https/etc
          return url;
        }}
        components={markdownComponents}
      >
        {formattedContent}
      </ReactMarkdown>

      {/* Copy Button */}
      {showCopyButton && (
        <div className="mt-2 flex justify-end">
          <button
            onClick={handleCopy}
            className="press-outline-button flex items-center gap-1.5 px-2 py-1 text-xs text-text-secondary"
            title="Copy to clipboard"
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-success" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>
      )}

      {/* Tool Call Cards appended at the bottom if provided */}
      {toolCalls && toolCalls.length > 0 && (
        <div className="mt-3 space-y-2">
          {toolCalls.map((tc) => (
            <ToolCallCard key={tc.id} toolCall={tc} defaultExpanded={false} />
          ))}
        </div>
      )}
    </div>
  );
};
