/**
 * ToolCallCard Component
 *
 * Displays a tool call with expand/collapse functionality.
 * Shows the tool name, status, and when expanded, the query/args and result.
 */

import { useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  Sparkles,
  Check,
  Loader2,
  AlertCircle,
} from '@/lib/lucide-icons';
import type { ToolCallInfo } from '../core/llm/types';

interface ToolCallCardProps {
  toolCall: ToolCallInfo;
  /** Start expanded (useful for in-progress calls) */
  defaultExpanded?: boolean;
}

/**
 * Format tool arguments for display
 */
const formatArgs = (args: Record<string, unknown>): string => {
  if (!args || Object.keys(args).length === 0) {
    return '';
  }

  // Special handling for Cypher queries
  if ('cypher' in args && typeof args.cypher === 'string') {
    let result = '';
    if ('query' in args && typeof args.query === 'string') {
      result += `Search: "${args.query}"\n\n`;
    }
    result += args.cypher;
    return result;
  }

  // Special handling for search/grep queries
  if ('query' in args && typeof args.query === 'string') {
    return args.query;
  }

  // For other tools, show as formatted JSON
  return JSON.stringify(args, null, 2);
};

/**
 * Get status icon and color
 */
const getStatusDisplay = (status: ToolCallInfo['status']) => {
  switch (status) {
    case 'running':
      return {
        icon: <Loader2 className="h-3.5 w-3.5 animate-spin" />,
        color: 'text-warning',
        bgColor: 'bg-base',
        borderColor: 'border-warning',
      };
    case 'completed':
      return {
        icon: <Check className="h-3.5 w-3.5" />,
        color: 'text-success',
        bgColor: 'bg-base',
        borderColor: 'border-success',
      };
    case 'error':
      return {
        icon: <AlertCircle className="h-3.5 w-3.5" />,
        color: 'text-error',
        bgColor: 'bg-base',
        borderColor: 'border-error',
      };
    default:
      return {
        icon: <Sparkles className="h-3.5 w-3.5" />,
        color: 'text-text-muted',
        bgColor: 'bg-surface',
        borderColor: 'border-border-subtle',
      };
  }
};

/**
 * Get a friendly display name for the tool
 */
const getToolDisplayName = (name: string): string => {
  const names: Record<string, string> = {
    search: '🔍 Search Code',
    cypher: '🔗 Cypher Query',
    grep: '🔎 Pattern Search',
    read: '📄 Read File',
    overview: '🗺️ Codebase Overview',
    explore: '🔬 Deep Dive',
    impact: '💥 Impact Analysis',
  };
  return names[name] || name;
};

export const ToolCallCard = ({ toolCall, defaultExpanded = false }: ToolCallCardProps) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const status = getStatusDisplay(toolCall.status);
  const formattedArgs = formatArgs(toolCall.args);

  return (
    <div
      className={`overflow-hidden rounded-xl border-2 ${status.borderColor} ${status.bgColor} transition-all`}
    >
      <div
        role="button"
        tabIndex={0}
        onClick={() => setIsExpanded(!isExpanded)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            setIsExpanded(!isExpanded);
          }
        }}
        className="flex w-full cursor-pointer items-center gap-2 px-3 py-3 text-left transition-colors select-none hover:bg-base/60"
      >
        <span className="text-text-muted">
          {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </span>

        <span className="flex-1 font-mono text-sm font-medium text-text-primary">
          {getToolDisplayName(toolCall.name)}
        </span>

        <span className={`press-badge flex items-center gap-1 border-current bg-transparent px-2 py-1 text-xs normal-case tracking-normal ${status.color}`}>
          {status.icon}
          <span className="capitalize">{toolCall.status}</span>
        </span>
      </div>

      {isExpanded && (
        <div className="border-t border-border-subtle/50">
          {formattedArgs && (
            <div className="border-b border-border-subtle/50 px-3 py-3">
              <div className="press-eyebrow mb-1.5 text-text-muted">
                {toolCall.name === 'cypher' ? 'Query' : 'Input'}
              </div>
              <pre className="overflow-x-auto rounded-lg border border-border-subtle bg-base p-3 font-mono text-xs whitespace-pre-wrap text-text-secondary">
                {formattedArgs}
              </pre>
            </div>
          )}

          {toolCall.result && (
            <div className="px-3 py-3">
              <div className="press-eyebrow mb-1.5 text-text-muted">
                Result
              </div>
              <div className="max-h-[400px] overflow-y-auto rounded-lg border border-border-subtle bg-base">
                <pre className="p-2 font-mono text-xs whitespace-pre-wrap text-text-secondary">
                  {toolCall.result.length > 3000
                    ? toolCall.result.slice(0, 3000) + '\n\n... (truncated)'
                    : toolCall.result}
                </pre>
              </div>
            </div>
          )}

          {toolCall.status === 'running' && !toolCall.result && (
            <div className="flex items-center gap-2 px-3 py-3 text-xs text-text-muted">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Executing...</span>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ToolCallCard;
