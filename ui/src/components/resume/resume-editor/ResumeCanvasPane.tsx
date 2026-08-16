import React, { RefObject } from 'react';
import { FileText, Loader2, Sparkles } from 'lucide-react';
import { renderEditorCanvasHTML } from '../../../utils/resumeRenderer';
import { getCleanTextFromDOM } from '../../../utils/resumeHelpers';

interface ResumeCanvasPaneProps {
  activeSubTab: 'editor' | 'pdf';
  fitToSinglePage: boolean;
  editorContent: string;
  onUpdateContent: (content: string) => void;
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
  editorContent,
  onUpdateContent,
  canvasParentRef,
  canvasScale,
  hasPDF,
  onUploadClick,
  onConvertToNewLayout,
  isConvertingLayout = false,
}) => {
  return (
    <div className="min-w-0 h-full flex flex-col glass rounded-xl border border-white/10 overflow-hidden resume-editor-pane shadow-2xl relative">
      {/* Canvas Sub-Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-white/5 bg-surface-100/70 flex-shrink-0 no-print">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span className="text-xs font-semibold text-gray-200">
            {activeSubTab === 'editor' ? 'Visual Resume Canvas' : 'Original Uploaded PDF Document'}
          </span>
          {fitToSinglePage && activeSubTab === 'editor' && (
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 font-medium">
              1-Page Fit Active
            </span>
          )}
        </div>

        <div className="flex items-center gap-3">
          {activeSubTab === 'pdf' && onConvertToNewLayout && (
            <button
              onClick={onConvertToNewLayout}
              disabled={isConvertingLayout}
              className="px-3 py-1.5 rounded-lg bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-emerald-600/30 disabled:opacity-50 transition-all"
              title="Convert original resume content directly into the standard 1-page layout"
            >
              {isConvertingLayout ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
              ) : (
                <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
              )}
              {isConvertingLayout ? 'Converting Layout...' : 'Convert to New Layout'}
            </button>
          )}
          <span className="text-xs font-mono text-gray-400">
            {editorContent.length.toLocaleString()} characters
          </span>
        </div>
      </div>

      {/* Printable Canvas & PDF Viewing Area */}
      <div 
        ref={canvasParentRef}
        className={`flex-1 overflow-y-auto overflow-x-hidden print-area bg-[#0b0f19] p-4 lg:p-6 flex justify-center ${activeSubTab === 'pdf' ? 'flex flex-col' : 'items-start'} relative`}
      >
        {activeSubTab === 'editor' ? (
          <div 
            id="resume-editor-canvas"
            contentEditable
            suppressContentEditableWarning
            onBlur={(e) => {
              const newText = getCleanTextFromDOM(e.currentTarget);
              if (newText !== editorContent) {
                onUpdateContent(newText);
              }
            }}
            dangerouslySetInnerHTML={{ 
              __html: renderEditorCanvasHTML(editorContent, fitToSinglePage) 
            }}
            className={`editor-textarea bg-white text-slate-800 shadow-2xl rounded-sm mx-auto flex-shrink-0 ${fitToSinglePage ? 'fit-page' : ''}`}
            style={{
              width: '8.5in',
              minHeight: '11in',
              transform: `scale(${canvasScale})`,
              transformOrigin: 'top center',
              marginBottom: canvasScale < 1 ? `calc((1 - ${canvasScale}) * -11in)` : '0px',
              outline: 'none',
              boxSizing: 'border-box'
            }}
            spellCheck
          />
        ) : (
          <div className="w-full h-full flex flex-col items-center justify-center p-2">
            {hasPDF ? (
              <iframe
                src="/api/v1/resume/active/pdf"
                className="w-full h-full border border-white/10 rounded-lg shadow-2xl bg-white"
                style={{
                  width: '100%',
                  height: '100%',
                  minHeight: '500px',
                  objectFit: 'contain'
                }}
                title="Original PDF Resume"
              />
            ) : (
              <div className="text-center p-8 max-w-sm glass rounded-xl border border-white/5">
                <FileText className="w-12 h-12 text-gray-600 mx-auto mb-3" />
                <h4 className="text-sm font-semibold text-white mb-2">No Original PDF Available</h4>
                <p className="text-xs text-gray-400 leading-relaxed mb-4">
                  To view your original PDF document here, please upload a PDF resume.
                </p>
                <button
                  onClick={onUploadClick}
                  className="btn-primary text-xs"
                >
                  Upload Resume PDF
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
