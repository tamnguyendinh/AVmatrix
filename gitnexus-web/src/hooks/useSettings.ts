import { useChatRuntime } from './chat-runtime/ChatRuntimeContext';

export const useSettings = () => {
  const { llmSettings, updateLLMSettings } = useChatRuntime();

  return {
    settings: llmSettings,
    updateSettings: updateLLMSettings,
  };
};
