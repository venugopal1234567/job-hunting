import { useState, useCallback } from 'react';
import { ChatMessage, ProposedEdit, GapQuestionPrompt, StructuredResume } from '../types';
import { chatWithResume, saveResumeContent, saveResumeVersion, revertResume, analyzeResume } from '../services/api';

let msgCounter = 0;
const mkId = () => `msg_${++msgCounter}_${Date.now()}`;

export const useResumeEditor = (jobId?: string) => {
  const [canvasStructured, setCanvasStructured] = useState<StructuredResume | null>(null);
  const [needsAnalysis, setNeedsAnalysis] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isChatLoading, setIsChatLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isReverting, setIsReverting] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [lastSavedSkills, setLastSavedSkills] = useState<string[]>([]);

  const initContent = useCallback((sr: StructuredResume | null, skills: string[]) => {
    setCanvasStructured(sr);
    setNeedsAnalysis(sr === null);
    setLastSavedSkills(skills);
    setIsDirty(false);
  }, []);

  const applyStructured = useCallback((sr: StructuredResume) => {
    setCanvasStructured(sr);
    setIsDirty(true);
  }, []);

  // Auto-save editor content to backend
  const saveContent = useCallback(async (structuredOverride?: StructuredResume) => {
    const structToSave = structuredOverride || canvasStructured;
    if (!structToSave || isSaving) return null;
    setIsSaving(true);
    try {
      const result = await saveResumeContent(structToSave);
      setLastSavedSkills(result.skills);
      setIsDirty(false);
      return result;
    } catch (err) {
      console.error('Save failed:', err);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, [canvasStructured, isSaving]);

  // Send a chat message to the AI
  const sendMessage = useCallback(async (userText: string, model?: string, customJd?: string, directCommand?: boolean) => {
    if (!userText.trim() || isChatLoading || !canvasStructured) return;

    const userMsg: ChatMessage = {
      id: mkId(),
      role: 'user',
      content: userText,
      timestamp: Date.now(),
    };
    setChatMessages(prev => [...prev, userMsg]);
    setIsChatLoading(true);

    try {
      const response = await chatWithResume(userText, canvasStructured, jobId, model, customJd, directCommand);

      let structRes = response.structured_resume;
      if (structRes) {
        const prevKeywords = canvasStructured?.highlight_keywords || [];
        const newKeywords = structRes.highlight_keywords || [];
        const mergedSet = new Set<string>();
        prevKeywords.forEach(k => k && mergedSet.add(k));
        newKeywords.forEach(k => k && mergedSet.add(k));
        structRes = {
          ...structRes,
          highlight_keywords: Array.from(mergedSet),
        };
      }

      const aiMsg: ChatMessage = {
        id: mkId(),
        role: 'assistant',
        content: response.message,
        proposedEdits: response.proposed_edits?.map((e: ProposedEdit) => ({ ...e })),
        gapPrompts: response.gap_prompts?.map((g: GapQuestionPrompt) => ({ ...g })),
        structuredResume: structRes,
        timestamp: Date.now(),
      };
      setChatMessages(prev => [...prev, aiMsg]);

      // If directCommand returned an updated structured resume, auto-apply to canvas
      if (directCommand && structRes) {
        applyStructured(structRes);
        saveContent(structRes);
      }
    } catch (err: any) {
      const errMsg: ChatMessage = {
        id: mkId(),
        role: 'assistant',
        content: `⚠️ Error: ${err.response?.data?.error || err.message || 'Chat failed'}`,
        timestamp: Date.now(),
      };
      setChatMessages(prev => [...prev, errMsg]);
    } finally {
      setIsChatLoading(false);
    }
  }, [canvasStructured, isChatLoading, jobId, applyStructured, saveContent]);

  // Answer a gap question — feed the answer back as a message
  const answerGapQuestion = useCallback((skill: string, answer: 'yes' | 'no' | string) => {
    const text = answer === 'yes'
      ? `Yes, I have experience with ${skill}. Please add it to my resume.`
      : answer === 'no'
      ? `No, I don't have experience with ${skill}.`
      : answer;
    sendMessage(text);
  }, [sendMessage]);

  // Save an "applied" version snapshot
  const saveAsApplied = useCallback(async (params: {
    jobId?: string;
    label: string;
    source: 'editor' | 'upload';
  }) => {
    if (!canvasStructured) return;
    await saveContent();
    return saveResumeVersion({
      snapshot_structured: canvasStructured,
      label: params.label,
      source: params.source,
    });
  }, [canvasStructured, saveContent]);

  // Revert edited text back to original
  const revertContent = useCallback(async () => {
    setIsReverting(true);
    try {
      const result = await revertResume();
      setCanvasStructured(result.structured);
      setNeedsAnalysis(result.structured === null);
      setIsDirty(false);
      return result;
    } catch (err) {
      console.error('Revert failed:', err);
      return null;
    } finally {
      setIsReverting(false);
    }
  }, []);

  const runAnalysis = useCallback(async () => {
    setIsAnalyzing(true);
    try {
      const result = await analyzeResume();
      setCanvasStructured(result.structured);
      setNeedsAnalysis(false);
    } catch (err) {
      console.error('Manual structuring failed:', err);
      alert('AI structuring failed to generate: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      setIsAnalyzing(false);
    }
  }, []);

  return {
    canvasStructured,
    needsAnalysis,
    isAnalyzing,
    chatMessages,
    isChatLoading,
    isSaving,
    isReverting,
    isDirty,
    lastSavedSkills,
    initContent,
    applyStructured,
    sendMessage,
    answerGapQuestion,
    saveContent,
    saveAsApplied,
    revertContent,
    runAnalysis,
    setChatMessages,
  };
};
