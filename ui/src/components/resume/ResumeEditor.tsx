import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Loader2, FileText } from 'lucide-react';
import { Job, StructuredResume } from '../../types';
import { getResumeContent, getActiveResume, analyzeJob, uploadResume, convertResumeToTemplate } from '../../services/api';
import { useResumeEditor } from '../../hooks/useResumeEditor';
import ChatPanel from './ChatPanel';
import AppliedDialog from './AppliedDialog';
import ResumeUploader from './ResumeUploader';
import { formatResumeTextToHTML } from '../../utils/resumeRenderer';
import { ResumeEditorHeader } from './resume-editor/ResumeEditorHeader';
import { ResumeCanvasPane } from './resume-editor/ResumeCanvasPane';

interface ResumeEditorProps {
  selectedJob?: Job | null;
}

const ResumeEditor: React.FC<ResumeEditorProps> = ({ selectedJob }) => {
  const {
    canvasStructured,
    needsAnalysis,
    isAnalyzing,
    chatMessages,
    isChatLoading,
    isSaving,
    isReverting,
    isDirty,
    initContent,
    applyStructured,
    sendMessage,
    answerGapQuestion,
    saveContent,
    saveAsApplied,
    revertContent,
    runAnalysis,
    setChatMessages,
  } = useResumeEditor(selectedJob?.id);

  const [loadingResume, setLoadingResume] = useState(true);
  const [noResume, setNoResume] = useState(false);
  const [atsScore, setAtsScore] = useState<number | null>(null);
  const [prevAtsScore, setPrevAtsScore] = useState<number | null>(null);
  const [atsLoading, setAtsLoading] = useState(false);
  const [chatVisible, setChatVisible] = useState(true);
  const [showAppliedDialog, setShowAppliedDialog] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');
  const [activeSubTab, setActiveSubTab] = useState<'editor' | 'pdf'>('editor');
  const [hasPDF, setHasPDF] = useState(false);
  const [activeModel, setActiveModel] = useState<string>('');
  const [customJdEnabled, setCustomJdEnabled] = useState(true);
  const [customJdText, setCustomJdText] = useState('');
  const [fitToSinglePage, setFitToSinglePage] = useState(true);
  const [exportingPDF] = useState(false);
  const [isConvertingLayout, setIsConvertingLayout] = useState(false);
  const [uploadPhase, setUploadPhase] = useState<'idle' | 'uploading' | 'analyzing' | 'done'>('idle');

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autoSaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const uploadInputRef = useRef<HTMLInputElement>(null);
  const canvasParentRef = useRef<HTMLDivElement>(null);
  const [canvasScale, setCanvasScale] = useState<number>(1);

  // Responsive scaling observer for the Visual Resume Canvas sheet
  useEffect(() => {
    const parentEl = canvasParentRef.current;
    if (!parentEl) return;

    const updateScale = () => {
      const pagePixelWidth = 816;
      const containerWidth = parentEl.clientWidth - 48;
      if (containerWidth > 0 && containerWidth < pagePixelWidth) {
        setCanvasScale(containerWidth / pagePixelWidth);
      } else {
        setCanvasScale(1);
      }
    };

    updateScale();
    const observer = new ResizeObserver(updateScale);
    observer.observe(parentEl);
    window.addEventListener('resize', updateScale);

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', updateScale);
    };
  }, []);

  // Load resume on mount
  useEffect(() => {
    const load = async () => {
      setLoadingResume(true);
      try {
        const [contentData, meta] = await Promise.all([
          getResumeContent(),
          getActiveResume(),
        ]);
        if (!contentData) {
          setNoResume(true);
          return;
        }
        initContent(contentData.structured, meta?.extracted_skills || []);
        setHasPDF(!!meta?.has_pdf);
        setNoResume(false);
      } catch {
        setNoResume(true);
      } finally {
        setLoadingResume(false);
      }
    };
    load();
  }, [initContent]);

  // Isolate chat messages on job change (Do NOT auto-run ATS)
  useEffect(() => {
    setChatMessages([]);
  }, [selectedJob?.id, setChatMessages]);

  const handleSave = useCallback(async (silent = false) => {
    const result = await saveContent();
    if (result) {
      if (!silent) {
        setSaveMessage('Saved!');
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2000);
      }
    }
  }, [saveContent]);

  // Debounced auto-save on content change
  useEffect(() => {
    if (!isDirty || !canvasStructured) return;
    if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    autoSaveTimerRef.current = setTimeout(() => {
      handleSave(true);
    }, 3000);
    return () => {
      if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    };
  }, [canvasStructured, isDirty, handleSave]);

  // ATS calculation is ONLY run when manually triggered by the user
  const runATS = useCallback(async (force = true) => {
    if (!selectedJob) return;
    setAtsLoading(true);
    try {
      const result = await analyzeJob(selectedJob.id, undefined, force, activeModel || undefined);
      setPrevAtsScore(atsScore);
      setAtsScore(result.ats_score);
    } catch {
      // silently ignore
    } finally {
      setAtsLoading(false);
    }
  }, [selectedJob, atsScore, activeModel]);

  const handleRevert = useCallback(async () => {
    if (!window.confirm('Are you sure you want to revert all changes? This will restore the original resume text.')) {
      return;
    }
    const result = await revertContent();
    if (result) {
      setChatMessages([]);
      setSaveMessage('Reverted to Original!');
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
    }
  }, [revertContent, setChatMessages]);

  const handleApplyFullResume = useCallback(async (replacement: string | object) => {
    if (typeof replacement === 'object' && replacement !== null) {
      const newStruct = { ...(replacement as StructuredResume) };
      const prevKeywords = canvasStructured?.highlight_keywords || [];
      const newKeywords = newStruct.highlight_keywords || [];
      const mergedSet = new Set<string>();
      prevKeywords.forEach(k => k && mergedSet.add(k));
      newKeywords.forEach(k => k && mergedSet.add(k));
      newStruct.highlight_keywords = Array.from(mergedSet);

      applyStructured(newStruct);
      await saveContent(newStruct);
    }
  }, [canvasStructured, applyStructured, saveContent]);

  const handleConvertToNewLayout = useCallback(async () => {
    setIsConvertingLayout(true);
    try {
      const result = await convertResumeToTemplate(undefined, activeModel, fitToSinglePage);
      if (result && result.parsed) {
        applyStructured(result.parsed);
        await saveContent(result.parsed);
        setActiveSubTab('editor');
        setSaveMessage('Converted to Standard Layout!');
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => setSaveMessage(''), 3000);
      }
    } catch (err: any) {
      alert('Failed to convert layout: ' + (err?.response?.data?.error || err.message || String(err)));
    } finally {
      setIsConvertingLayout(false);
    }
  }, [activeModel, fitToSinglePage, applyStructured, saveContent]);

  const handleExportPDF = () => {
    if (!canvasStructured) return;
    const fullHTML = formatResumeTextToHTML(canvasStructured, fitToSinglePage);

    const printFrame = document.createElement('iframe');
    printFrame.style.position = 'fixed';
    printFrame.style.right = '0';
    printFrame.style.bottom = '0';
    printFrame.style.width = '0';
    printFrame.style.height = '0';
    printFrame.style.border = '0';
    document.body.appendChild(printFrame);

    const frameDoc = printFrame.contentWindow?.document;
    if (frameDoc) {
      frameDoc.open();
      frameDoc.write(fullHTML);
      frameDoc.close();
      setTimeout(() => {
        printFrame.contentWindow?.focus();
        printFrame.contentWindow?.print();
        setTimeout(() => {
          document.body.removeChild(printFrame);
        }, 1000);
      }, 250);
    }
  };

  const handleResumeUploaded = async () => {
    setLoadingResume(true);
    try {
      const [contentData, meta] = await Promise.all([getResumeContent(), getActiveResume()]);
      if (contentData) {
        // Only initialize canvas if there was no active resume loaded
        if (!canvasStructured && contentData.structured) {
          initContent(contentData.structured, meta?.extracted_skills || []);
        }
        setHasPDF(!!meta?.has_pdf);
        setNoResume(false);
      }
    } finally {
      setLoadingResume(false);
      setUploadPhase('idle');
    }
  };

  const handleDirectUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setLoadingResume(true);
    setUploadPhase('uploading');
    try {
      // Step 1: Upload and wait for backend analysis
      setUploadPhase('analyzing');
      await uploadResume(file);
      await handleResumeUploaded();
      setSaveMessage('Base PDF updated! Working canvas preserved.');
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => setSaveMessage(''), 3500);
    } catch (err) {
      console.error(err);
      alert('Upload failed: ' + (err instanceof Error ? err.message : String(err)));
      setUploadPhase('idle');
      setLoadingResume(false);
    } finally {
      e.target.value = '';
    }
  };

  if (loadingResume) {
    return (
      <div className="flex flex-col items-center justify-center h-full min-h-[400px] gap-3">
        <Loader2 className="w-6 h-6 text-brand-400 animate-spin" />
        <p className="text-sm text-gray-400">
          {uploadPhase === 'uploading' ? 'Uploading your resume...' :
           uploadPhase === 'analyzing' ? 'Analyzing your resume...' :
           'Loading resume...'}
        </p>
      </div>
    );
  }

  if (noResume) {
    return (
      <div className="max-w-xl mx-auto py-10">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-surface-200 border border-white/5 flex items-center justify-center mx-auto mb-4">
            <FileText className="w-8 h-8 text-gray-600" />
          </div>
          <h2 className="text-lg font-bold text-white">Upload Your Resume</h2>
          <p className="text-sm text-gray-400 mt-2">
            Upload your resume to start using the AI editor.
          </p>
        </div>
        <div className="glass rounded-xl p-6">
          <ResumeUploader onResumeUploaded={handleResumeUploaded} />
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <input
        type="file"
        ref={uploadInputRef}
        style={{ display: 'none' }}
        accept=".pdf,.txt"
        onChange={handleDirectUpload}
      />

      <ResumeEditorHeader
        selectedJob={selectedJob}
        atsScore={atsScore}
        prevAtsScore={prevAtsScore}
        atsLoading={atsLoading}
        onReanalyzeATS={runATS}
        saveMessage={saveMessage}
        activeSubTab={activeSubTab}
        onSubTabChange={setActiveSubTab}
        fitToSinglePage={fitToSinglePage}
        onToggleFitToSinglePage={() => setFitToSinglePage(!fitToSinglePage)}
        isSaving={isSaving}
        isReverting={isReverting}
        onRevert={handleRevert}
        exportingPDF={exportingPDF}
        onExportPDF={handleExportPDF}
        chatVisible={chatVisible}
        onToggleChat={() => setChatVisible(!chatVisible)}
        onUploadClick={() => uploadInputRef.current?.click()}
        onConvertToNewLayout={handleConvertToNewLayout}
        isConvertingLayout={isConvertingLayout}
      />

      {/* 50/50 Split Workspace Grid */}
      <div className={`grid grid-cols-1 ${chatVisible ? 'lg:grid-cols-2' : 'lg:grid-cols-1'} gap-4 h-[calc(100vh-210px)] min-h-[500px] w-full no-print`}>
        {/* AI Resume Coach Sidebar Panel */}
        {chatVisible && (
          <div className="min-w-0 h-full glass rounded-xl border border-white/10 overflow-hidden flex flex-col no-print shadow-xl">
            <ChatPanel
              messages={chatMessages}
              loading={isChatLoading}
              onSend={(text, customJd, directCommand) => sendMessage(text, activeModel, customJd, directCommand)}
              onAnswerGap={answerGapQuestion}
              jobTitle={selectedJob?.title}
              activeModel={activeModel}
              onModelChange={setActiveModel}
              onApplyFullResume={handleApplyFullResume}
              customJdText={customJdText}
              customJdEnabled={customJdEnabled}
              onCustomJdTextChange={setCustomJdText}
              onCustomJdEnabledChange={setCustomJdEnabled}
            />
          </div>
        )}

        {/* Visual Resume Canvas / Original PDF Viewer */}
        <ResumeCanvasPane
          activeSubTab={activeSubTab}
          fitToSinglePage={fitToSinglePage}
          canvasStructured={canvasStructured}
          needsAnalysis={needsAnalysis}
          onAnalyze={runAnalysis}
          isAnalyzing={isAnalyzing}
          canvasParentRef={canvasParentRef}
          canvasScale={canvasScale}
          hasPDF={hasPDF}
          onUploadClick={() => uploadInputRef.current?.click()}
          onConvertToNewLayout={handleConvertToNewLayout}
          isConvertingLayout={isConvertingLayout}
        />
      </div>

      {/* Applied Dialog */}
      <AppliedDialog
        isOpen={showAppliedDialog}
        onClose={() => setShowAppliedDialog(false)}
        canvasStructured={canvasStructured}
        selectedJob={selectedJob}
        onSaveVersion={saveAsApplied}
        onResumeUploaded={handleResumeUploaded}
      />
    </div>
  );
};

export default ResumeEditor;
