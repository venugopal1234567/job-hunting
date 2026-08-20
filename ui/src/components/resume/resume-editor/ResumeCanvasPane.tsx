import React, { RefObject, useEffect, useRef } from 'react';
import { FileText, Loader2, Sparkles } from 'lucide-react';
import { renderFromStructured } from '../../../utils/resumeRenderer';
import { StructuredResume } from '../../../types';

interface ResumeCanvasPaneProps {
  activeSubTab: 'editor' | 'pdf';
  fitToSinglePage: boolean;
  canvasStructured: StructuredResume | null;
  needsAnalysis: boolean;
  onAnalyze: () => void;
  isAnalyzing: boolean;
  canvasParentRef: RefObject<HTMLDivElement>;
  canvasScale: number;
  hasPDF: boolean;
  onUploadClick: () => void;
  onConvertToNewLayout?: () => void;
  isConvertingLayout?: boolean;
}

export const ResumeCanvasPane: React.FC<ResumeCanvasPaneProps> = ({
  activeSubTab,
  fitToSinglePage,
  canvasStructured,
  needsAnalysis,
  onAnalyze,
  isAnalyzing,
  canvasParentRef,
  canvasScale,
  hasPDF,
  onUploadClick,
  onConvertToNewLayout,
  isConvertingLayout = false,
}) => {
  const canvasRef = useRef<HTMLDivElement>(null);

  // Track the last (content + fitToSinglePage) pair we rendered
  const lastRenderedKey = useRef<string>('');

  useEffect(() => {
    if (activeSubTab !== 'editor') {
      lastRenderedKey.current = '';
      return;
    }

    const canvas = canvasRef.current;
    if (!canvas || !canvasStructured) return;

    const newKey = `${JSON.stringify(canvasStructured)}::${fitToSinglePage}`;
    if (newKey === lastRenderedKey.current) return;
    lastRenderedKey.current = newKey;

    canvas.innerHTML = renderFromStructured(canvasStructured, fitToSinglePage);

    return () => {
      if (canvas) {
        canvas.innerHTML = '';
      }
    };
  }, [activeSubTab, canvasStructured, fitToSinglePage]);

  // Compute character length safely
  const charLength = canvasStructured ? JSON.stringify(canvasStructured).length : 0;

  return (
    <div className="min-w-0 h-full flex flex-col bg-surface-container-lowest rounded-2xl border border-surface-variant overflow-hidden shadow-elevation-1 relative">
      {/* Canvas Sub-Header: only displayed in visual editor tab */}
      {activeSubTab === 'editor' && (
        <div className="flex items-center justify-between px-4 py-3 border-b border-surface-variant bg-surface-container-low/50 flex-shrink-0 no-print">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500"></span>
            <span className="text-xs font-bold font-headline text-on-surface">
              Visual Resume Canvas
            </span>
            {fitToSinglePage && (
              <span className="text-[10px] px-2.5 py-0.5 rounded-full bg-primary-fixed text-on-primary-fixed border border-primary-fixed-dim/50 font-bold">
                1-Page Fit Active
              </span>
            )}
          </div>

          <div className="flex items-center gap-3">
            <span className="text-xs font-mono text-on-surface-variant font-medium">
              {charLength.toLocaleString()} characters
            </span>
          </div>
        </div>
      )}

      {/* Printable Canvas & PDF Viewing Area */}
      <div
        ref={canvasParentRef}
        className={`flex-1 ${activeSubTab === 'pdf' ? 'p-0 flex flex-col overflow-hidden h-full min-h-0' : 'p-4 lg:p-6 flex justify-center items-start overflow-y-auto overflow-x-hidden'} bg-surface-container-low relative`}
      >
        {activeSubTab === 'editor' ? (
          needsAnalysis ? (
            <div key="editor-needs-analysis" className="flex flex-col items-center justify-center h-full gap-4 text-center max-w-sm mx-auto p-6 bg-surface-container-lowest rounded-2xl border border-surface-variant shadow-sm">
              <div className="w-12 h-12 rounded-full bg-primary-fixed text-primary flex items-center justify-center">
                <FileText className="w-6 h-6" />
              </div>
              <h4 className="text-sm font-bold font-headline text-on-surface">Resume Could Not Be Structured Automatically</h4>
              <p className="text-xs text-on-surface-variant leading-relaxed">
                We've stored your raw resume text safely. Click the button below to retry generating a structured visual layout with AI.
              </p>
              <button
                onClick={onAnalyze}
                disabled={isAnalyzing}
                className="btn-primary text-xs flex items-center gap-1.5"
              >
                {isAnalyzing ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Sparkles className="w-4 h-4 text-white" />
                )}
                {isAnalyzing ? 'Analyzing Resume...' : 'Generate Structure'}
              </button>
            </div>
          ) : (
            <div
              key="resume-editor-canvas-container"
              ref={canvasRef}
              id="resume-editor-canvas"
              className={`editor-textarea bg-white text-slate-800 shadow-elevation-2 rounded-sm mx-auto flex-shrink-0 ${fitToSinglePage ? 'fit-page' : ''}`}
              style={{
                width: '8.5in',
                minHeight: '11in',
                transform: `scale(${canvasScale})`,
                transformOrigin: 'top center',
                marginBottom: canvasScale < 1 ? `calc((1 - ${canvasScale}) * -11in)` : '0px',
                outline: 'none',
                boxSizing: 'border-box',
              }}
            />
          )
        ) : (
          <div key="resume-pdf-viewer-container" className="w-full h-full flex-1 min-h-0 bg-white relative block">
            <iframe
              key="original-pdf-iframe"
              src="/api/v1/resume/active/pdf#toolbar=1&view=FitH"
              className="w-full h-full border-0 bg-white block"
              style={{
                width: '100%',
                height: '100%',
                border: 'none',
                display: 'block',
              }}
              title="Original PDF Resume"
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default ResumeCanvasPane;
