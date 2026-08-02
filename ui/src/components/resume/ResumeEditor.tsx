import React, { useEffect, useState, useRef, useCallback } from 'react';
import {
  Download, Save, Upload, Clock, PanelRight, PanelRightClose,
  FileText, AlertCircle, Loader2, CheckCircle2, GitBranch,
  Sparkles, Cpu, Zap
} from 'lucide-react';
import { Job, ResumeVersion } from '../../types';
import { getResumeFullText, getActiveResume, analyzeJob, getResumeVersions, getVersionText, uploadResume } from '../../services/api';
import { useResumeEditor } from '../../hooks/useResumeEditor';
import ChatPanel from './ChatPanel';
import ATSScoreBar from './ATSScoreBar';
import AppliedDialog from './AppliedDialog';
import ResumeUploader from './ResumeUploader';

// Helper to parse plain text resume into highly styled, beautiful HTML representation
// Build a complete, self-contained HTML doc for high-fidelity PDF printing
const generatePrintHTML = (text: string): string => {
  if (!text) return '';

  // ── Pre-process: merge isolated bullet chars with the next text line ────────
  const rawLines = text.split('\n');
  const mergedLines: string[] = [];
  for (let i = 0; i < rawLines.length; i++) {
    const t = rawLines[i].trim().replace(/[\u200b\u200c\u200d\ufeff]/g, '');
    const isBulletOnly = t === '•' || t === '-' || t === '*' || t === '▪' || t === '◦';
    if (isBulletOnly && i + 1 < rawLines.length) {
      let next = i + 1;
      while (next < rawLines.length && rawLines[next].trim() === '') next++;
      if (next < rawLines.length) {
        mergedLines.push('• ' + rawLines[next].trim());
        i = next;
        continue;
      }
    }
    mergedLines.push(rawLines[i]);
  }

  // ── Render lines to professional HTML body ───────────────────────────────────
  const dateRx = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[\s\-–]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))\s*$/i;
  const sectionRx = /^(CAREER SUMMARY|SUMMARY|TECHNICAL SKILLS|SKILLS|PROFESSIONAL EXPERIENCE|EXPERIENCE|WORK EXPERIENCE|PROJECTS|EDUCATION|CERTIFICATIONS|ORGANIZATIONS|ADDITIONAL INFORMATION|LANGUAGES)$/i;

  let body = '';
  let pastHeader = false;

  for (let i = 0; i < mergedLines.length; i++) {
    const line = mergedLines[i];
    const t = line.trim();
    if (!t) continue;

    // Section header
    const isSection = sectionRx.test(t) ||
      (t.length < 40 && t === t.toUpperCase() && !/[@|+]/.test(t) && !/^[0-9\-–\s|/]+$/.test(t) && t.length > 3);

    if (isSection) {
      pastHeader = true;
      body += `<h2>${t}</h2>\n`;
      continue;
    }

    // Name / title / contact (before first section)
    if (!pastHeader) {
      if (i === 0) {
        body += `<h1>${t}</h1>\n`;
      } else if (/[@|+]/.test(t) || /\d{5,}/.test(t)) {
        body += `<p class="contact">${t}</p>\n`;
      } else {
        body += `<p class="subtitle">${t}</p>\n`;
      }
      continue;
    }

    // Job title + date on same line
    const dm = t.match(dateRx);
    if (dm && t.length < 140) {
      const date = dm[1];
      const left = t.replace(dateRx, '').replace(/[\s|–—-]+$/, '').trim();
      if (left) {
        body += `<div class="job-row"><span class="job-title">${left}</span><span class="job-date">${date}</span></div>\n`;
        continue;
      }
    }

    // Bullet
    const isBullet = /^[•\-*▪◦]/.test(t);
    if (isBullet) {
      const txt = t.replace(/^[•\-*▪◦\s\u200b]+/, '').trim();
      body += `<p class="bullet">• ${txt}</p>\n`;
      continue;
    }

    // Sub-headers (company | location, role titles)
    if ((t.includes('|') || t.includes(' – ') || t.includes(' - ')) && t.length < 120) {
      body += `<p class="company">${t}</p>\n`;
      continue;
    }

    // Regular paragraph
    body += `<p class="body-text">${t}</p>\n`;
  }

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Resume</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Calibri:wght@400;700&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    @page {
      size: letter;
      margin: 0.6in 0.65in;
    }

    body {
      font-family: 'Calibri', 'Inter', 'Georgia', sans-serif;
      font-size: 10.5pt;
      color: #1a1a1a;
      background: #fff;
      line-height: 1.35;
    }

    h1 {
      font-family: inherit;
      font-size: 20pt;
      font-weight: 700;
      color: #1a1a2e;
      text-align: center;
      margin-bottom: 2pt;
      letter-spacing: 0.01em;
    }

    .subtitle {
      font-size: 10.5pt;
      font-weight: 600;
      color: #444;
      text-align: center;
      margin-bottom: 2pt;
    }

    .contact {
      font-size: 9.5pt;
      color: #333;
      text-align: center;
      margin-bottom: 6pt;
    }

    h2 {
      font-family: inherit;
      font-size: 10.5pt;
      font-weight: 700;
      color: #1a1a2e;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      border-bottom: 0.75pt solid #aaa;
      margin-top: 10pt;
      margin-bottom: 4pt;
      padding-bottom: 1.5pt;
    }

    .job-row {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      margin-top: 5pt;
      margin-bottom: 1.5pt;
    }
    .job-title {
      font-size: 10.5pt;
      font-weight: 700;
      color: #1a1a2e;
    }
    .job-date {
      font-size: 9.5pt;
      font-weight: 400;
      color: #555;
    }

    .company {
      font-size: 10.5pt;
      font-weight: 600;
      color: #1a1a2e;
      margin-bottom: 2pt;
    }

    .bullet {
      font-size: 10pt;
      color: #1a1a1a;
      line-height: 1.35;
      padding-left: 14pt;
      text-indent: -14pt;
      text-align: justify;
      margin-bottom: 2pt;
    }

    .body-text {
      font-size: 10pt;
      color: #1a1a1a;
      line-height: 1.4;
      text-align: justify;
      margin-bottom: 2pt;
    }

    @media print {
      body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
      h2 { border-bottom-color: #999; }
    }
  </style>
</head>
<body>
${body}
</body>
</html>`;
};


const formatResumeTextToHTML = (text: string): string => {
  if (!text) return '';

  // Step 1: Pre-process lines to merge separate bullet characters with their text
  const rawLines = text.split('\n');
  const lines: string[] = [];

  for (let i = 0; i < rawLines.length; i++) {
    const currentLine = rawLines[i];
    const currentTrimmed = currentLine.trim();

    // Clean zero-width chars and spaces to identify pure bullet indicators
    const cleaned = currentTrimmed.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
    const isBulletChar = cleaned === '•' || cleaned === '-' || cleaned === '*' || cleaned === '▪' || cleaned === '◦';

    if (isBulletChar && i + 1 < rawLines.length) {
      // Merge with the next non-empty line
      let nextIdx = i + 1;
      while (nextIdx < rawLines.length && rawLines[nextIdx].trim() === '') {
        nextIdx++;
      }
      if (nextIdx < rawLines.length) {
        // Merge the bullet and text
        lines.push('• ' + rawLines[nextIdx].trim());
        i = nextIdx; // Skip to the merged line
        continue;
      }
    }
    lines.push(currentLine);
  }

  // Step 2: Format lines as styled HTML elements
  let htmlLines: string[] = [];
  let foundFirstHeader = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    if (trimmed === '') {
      // Avoid consecutive empty spacers
      if (i > 0 && lines[i - 1].trim() === '') {
        continue;
      }
      htmlLines.push('<div style="height: 5px;"></div>');
      continue;
    }

    // Check if it's a major section header
    const isSectionHeader = /^(CAREER SUMMARY|SUMMARY|TECHNICAL SKILLS|SKILLS|PROFESSIONAL EXPERIENCE|EXPERIENCE|WORK EXPERIENCE|PROJECTS|EDUCATION|CERTIFICATIONS|ORGANIZATIONS|ADDITIONAL INFORMATION|LANGUAGES)$/i.test(trimmed)
      || (trimmed.length < 40 && trimmed === trimmed.toUpperCase() && !trimmed.includes('|') && !trimmed.includes('@') && !trimmed.includes('+') && !/^[0-9\-–\s|/]+$/.test(trimmed) && trimmed.length > 3);

    if (isSectionHeader) {
      foundFirstHeader = true;
      htmlLines.push(
        `<h2 style="font-size: 11.5px; font-weight: 700; color: #1e3a8a; border-bottom: 0.75px solid #cbd5e1; margin-top: 15px; margin-bottom: 5px; text-transform: uppercase; letter-spacing: 0.05em; padding-bottom: 2px; font-family: 'Inter', sans-serif;">${trimmed}</h2>`
      );
      continue;
    }

    // Header section before the first section header (Name, Title, Contact)
    if (!foundFirstHeader) {
      if (i === 0 || (i === 1 && lines[0].trim() === '')) {
        // Candidate Name
        htmlLines.push(
          `<h1 style="font-size: 20px; font-weight: 700; color: #1e3a8a; text-align: center; margin-bottom: 3px; letter-spacing: -0.01em; font-family: 'Inter', sans-serif;">${trimmed}</h1>`
        );
      } else if (trimmed.includes('@') || trimmed.includes('|') || trimmed.includes('+') || /\d{5,}/.test(trimmed)) {
        // Contact details
        htmlLines.push(
          `<p style="font-size: 10px; color: #334155; text-align: center; margin-bottom: 3px; line-height: 1.3; font-family: 'Inter', sans-serif;">${trimmed}</p>`
        );
      } else {
        // Subtitle / Job Title
        htmlLines.push(
          `<p style="font-size: 12px; font-weight: 600; color: #475569; text-align: center; margin-bottom: 8px; font-family: 'Inter', sans-serif;">${trimmed}</p>`
        );
      }
      continue;
    }

    // Check if the line contains a date range (e.g. Jun 2023 - Present, 2015 - 2019, Jun 2021 - May 2023)
    const dateRegex = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\u2013\u2014]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))\s*$/i;
    const hasDateRange = dateRegex.test(trimmed);

    if (hasDateRange && trimmed.length < 120) {
      const match = trimmed.match(dateRegex);
      if (match) {
        const dateText = match[1];
        const leftText = trimmed.replace(dateRegex, '').trim().replace(/[\s|–—-]+$/, '').trim();
        
        if (leftText) {
          htmlLines.push(
            `<div style="display: flex; justify-content: space-between; font-size: 11px; font-weight: 700; color: #1e293b; margin-top: 6px; margin-bottom: 2px; font-family: 'Inter', sans-serif;">` +
              `<span>${leftText}</span>` +
              `<span style="font-weight: 600; color: #475569;">${dateText}</span>` +
            `</div>`
          );
          continue;
        }
      }
    }

    // Check if it's a bullet point line
    const isBullet = trimmed.startsWith('•') || trimmed.startsWith('-') || trimmed.startsWith('*') || trimmed.startsWith('▪') || trimmed.startsWith('◦');
    if (isBullet) {
      // Strips ALL leading bullets, zero-width chars, and spaces to prevent double bullets
      const cleanBulletText = trimmed.replace(/^[•\-*▪◦\s\u200b\ufeff]+/g, '').trim();
      htmlLines.push(
        `<p style="font-size: 11px; line-height: 1.45; color: #334155; margin-bottom: 2.5px; padding-left: 14px; text-indent: -14px; text-align: justify; font-family: 'Inter', sans-serif;">• ${cleanBulletText}</p>`
      );
      continue;
    }

    // Check if it's a professional experience subheader (like company name)
    const isSubHeader = trimmed.includes('|') || trimmed.includes(' - ') || trimmed.includes('–') || /\b(19|20)\d{2}\b/.test(trimmed);
    if (isSubHeader && trimmed.length < 120) {
      htmlLines.push(
        `<p style="font-size: 11px; font-weight: 700; color: #1e3a8a; margin-top: 8px; margin-bottom: 2px; font-family: 'Inter', sans-serif;">${trimmed}</p>`
      );
    } else {
      // General body paragraph
      htmlLines.push(
        `<p style="font-size: 11px; line-height: 1.45; color: #334155; margin-bottom: 3px; text-align: justify; font-family: 'Inter', sans-serif;">${trimmed}</p>`
      );
    }
  }

  return htmlLines.join('\n');
};

interface ResumeEditorProps {
  selectedJob?: Job | null;
}

const ResumeEditor: React.FC<ResumeEditorProps> = ({ selectedJob }) => {
  const {
    editorContent,
    trackedChanges,
    pendingChanges,
    chatMessages,
    isChatLoading,
    isSaving,
    isDirty,
    lastSavedSkills,
    initContent,
    updateContent,
    sendMessage,
    answerGapQuestion,
    acceptChange,
    rejectChange,
    saveContent,
    saveAsApplied,
    setChatMessages,
  } = useResumeEditor(selectedJob?.id);

  const [loadingResume, setLoadingResume] = useState(true);
  const [noResume, setNoResume] = useState(false);
  const [atsScore, setAtsScore] = useState<number | null>(null);
  const [prevAtsScore, setPrevAtsScore] = useState<number | null>(null);
  const [atsLoading, setAtsLoading] = useState(false);
  const [chatVisible, setChatVisible] = useState(true);
  const [showAppliedDialog, setShowAppliedDialog] = useState(false);
  const [showVersions, setShowVersions] = useState(false);
  const [versions, setVersions] = useState<ResumeVersion[]>([]);
  const [loadingVersions, setLoadingVersions] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');
  const [activeSubTab, setActiveSubTab] = useState<'editor' | 'pdf'>('editor');
  const [hasPDF, setHasPDF] = useState(false);
  
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autoSaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const uploadInputRef = useRef<HTMLInputElement>(null);

  // Load resume on mount
  useEffect(() => {
    const load = async () => {
      setLoadingResume(true);
      try {
        const [textData, meta] = await Promise.all([
          getResumeFullText(),
          getActiveResume(),
        ]);
        if (!textData) {
          setNoResume(true);
          return;
        }
        initContent(textData.text, meta?.extracted_skills || []);
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

  // Auto-run ATS when job changes
  useEffect(() => {
    if (selectedJob && editorContent) {
      runATS();
    }
  }, [selectedJob?.id]);

  // Debounced auto-save on content change
  useEffect(() => {
    if (!isDirty || !editorContent) return;
    if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    autoSaveTimerRef.current = setTimeout(() => {
      handleSave(true);
    }, 3000);
    return () => {
      if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    };
  }, [editorContent, isDirty]);

  const runATS = useCallback(async () => {
    if (!selectedJob) return;
    setAtsLoading(true);
    try {
      const result = await analyzeJob(selectedJob.id);
      setPrevAtsScore(atsScore);
      setAtsScore(result.ats_score);
    } catch {
      // silently ignore
    } finally {
      setAtsLoading(false);
    }
  }, [selectedJob, atsScore]);

  const handleSave = useCallback(async (silent = false) => {
    const result = await saveContent();
    if (result) {
      if (!silent) {
        setSaveMessage('Saved!');
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2000);
      }
      // Re-run ATS after saving
      if (selectedJob) runATS();
    }
  }, [saveContent, selectedJob, runATS]);

  const handleDownloadPDF = () => {
    const resumeHTML = generatePrintHTML(editorContent);
    const printWin = window.open('', '_blank', 'width=900,height=700');
    if (!printWin) return;
    printWin.document.open();
    printWin.document.write(resumeHTML);
    printWin.document.close();
    // Give the browser a moment to render fonts/styles before triggering print
    printWin.onload = () => {
      setTimeout(() => {
        printWin.focus();
        printWin.print();
      }, 400);
    };
  };

  const loadVersions = async () => {
    setLoadingVersions(true);
    try {
      const v = await getResumeVersions();
      setVersions(v || []);
    } finally {
      setLoadingVersions(false);
    }
  };

  const loadVersionText = async (versionId: string) => {
    const text = await getVersionText(versionId);
    if (text) {
      updateContent(text);
    }
  };

  const handleResumeUploaded = async () => {
    setLoadingResume(true);
    try {
      const [textData, meta] = await Promise.all([getResumeFullText(), getActiveResume()]);
      if (textData) {
        initContent(textData.text, meta?.extracted_skills || []);
        setHasPDF(!!meta?.has_pdf);
        setNoResume(false);
      }
    } finally {
      setLoadingResume(false);
    }
  };

  const handleDirectUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setLoadingResume(true);
    try {
      await uploadResume(file);
      await handleResumeUploaded();
    } catch (err) {
      console.error(err);
      alert('Upload failed: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      setLoadingResume(false);
      e.target.value = '';
    }
  };

  // ── No resume state ──────────────────────────────────────────────────────────
  if (loadingResume) {
    return (
      <div className="flex items-center justify-center h-full min-h-[400px] gap-3">
        <Loader2 className="w-6 h-6 text-brand-400 animate-spin" />
        <p className="text-sm text-gray-400">Loading resume...</p>
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
          <ResumeUploader />
        </div>
      </div>
    );
  }

  // ── Main Editor ─────────────────────────────────────────────────────────────
  return (
    <>
      {/* Top Toolbar */}
      <div className="flex flex-wrap items-center gap-3 mb-4 no-print">
        {/* ATS Score */}
        <div className="flex-1 min-w-[280px]">
          <ATSScoreBar
            score={atsScore}
            previousScore={prevAtsScore}
            loading={atsLoading}
            jobTitle={selectedJob?.title}
            pendingChanges={pendingChanges.length}
            onReanalyze={selectedJob ? runATS : undefined}
          />
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {/* Save indicator */}
          {saveMessage && (
            <span className="flex items-center gap-1 text-xs text-emerald-400 animate-fade-in">
              <CheckCircle2 className="w-3.5 h-3.5" /> {saveMessage}
            </span>
          )}
          {isDirty && !saveMessage && (
            <span className="text-xs text-amber-400 animate-pulse">Unsaved changes</span>
          )}

          <input
            type="file"
            ref={uploadInputRef}
            style={{ display: 'none' }}
            accept=".pdf,.txt"
            onChange={handleDirectUpload}
          />

          <button
            id="btn-direct-upload"
            onClick={() => uploadInputRef.current?.click()}
            className="btn-ghost text-xs flex items-center gap-1.5"
            title="Upload a new resume PDF/TXT"
          >
            <Upload className="w-3.5 h-3.5" />
            Upload New
          </button>

          <button
            id="btn-save-resume"
            onClick={() => handleSave(false)}
            disabled={isSaving || !isDirty}
            className="btn-ghost text-xs flex items-center gap-1.5 disabled:opacity-40"
          >
            {isSaving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5" />}
            Save
          </button>

          <button
            id="btn-mark-applied"
            onClick={() => setShowAppliedDialog(true)}
            className="btn-ghost text-xs flex items-center gap-1.5 text-brand-400 hover:text-brand-300"
          >
            <CheckCircle2 className="w-3.5 h-3.5" />
            Applied?
          </button>

          <button
            id="btn-version-history"
            onClick={() => { setShowVersions(!showVersions); if (!showVersions) loadVersions(); }}
            className={`btn-ghost text-xs flex items-center gap-1.5 ${showVersions ? 'text-brand-400' : ''}`}
          >
            <GitBranch className="w-3.5 h-3.5" />
            History
          </button>

          <button
            id="btn-download-pdf"
            onClick={handleDownloadPDF}
            className="btn-primary text-xs flex items-center gap-1.5"
          >
            <Download className="w-3.5 h-3.5" />
            PDF
          </button>

          <button
            id="btn-toggle-chat"
            onClick={() => setChatVisible(!chatVisible)}
            className="btn-ghost p-2"
            title={chatVisible ? 'Hide AI Chat' : 'Show AI Chat'}
          >
            {chatVisible
              ? <PanelRightClose className="w-4 h-4" />
              : <PanelRight className="w-4 h-4" />
            }
          </button>
        </div>
      </div>

      {/* Version History Drawer */}
      {showVersions && (
        <div className="glass rounded-xl border border-white/5 mb-4 p-4 no-print animate-fade-in">
          <div className="flex items-center justify-between mb-3">
            <p className="text-xs font-semibold text-white flex items-center gap-1.5">
              <GitBranch className="w-3.5 h-3.5 text-brand-400" />
              Resume Version History
            </p>
          </div>
          {loadingVersions ? (
            <div className="flex items-center gap-2 py-2">
              <Loader2 className="w-4 h-4 text-brand-400 animate-spin" />
              <span className="text-xs text-gray-500">Loading...</span>
            </div>
          ) : versions.length === 0 ? (
            <p className="text-xs text-gray-600 py-2">No saved versions yet. Click "Applied?" to save a snapshot.</p>
          ) : (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {versions.map(v => (
                <div key={v.id} className="flex items-center gap-3 p-2.5 rounded-lg bg-surface-200 hover:bg-surface-300 transition-all cursor-pointer group"
                  onClick={() => { loadVersionText(v.id); setShowVersions(false); }}
                >
                  <div className={`w-2 h-2 rounded-full flex-shrink-0 ${v.source === 'upload' ? 'bg-brand-400' : 'bg-emerald-400'}`} />
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium text-white truncate">{v.label || 'Unnamed snapshot'}</p>
                    <p className="text-[10px] text-gray-500">
                      {v.source === 'upload' ? '📎 Uploaded' : '✏️ Edited'} ·{' '}
                      {new Date(v.applied_at).toLocaleDateString()}
                      {v.job_title && ` · ${v.job_title}`}
                    </p>
                  </div>
                  <span className="text-[10px] text-brand-400 opacity-0 group-hover:opacity-100 transition-opacity">
                    Restore →
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Split Pane: Editor + Chat */}
      <div className={`flex gap-4 h-[calc(100vh-280px)] min-h-[500px]`}>

        {/* ── Editor Pane ─────────────────────────────────────────────────── */}
        <div className={`flex flex-col ${chatVisible ? 'flex-[1.6]' : 'flex-1'} min-w-0 transition-all duration-300`}>
          {/* Pending changes bar */}
          {pendingChanges.length > 0 && (
            <div className="mb-2 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-between gap-3 no-print">
              <div className="flex items-center gap-2">
                <Sparkles className="w-3.5 h-3.5 text-amber-400" />
                <span className="text-xs text-amber-300">
                  {pendingChanges.length} AI suggestion{pendingChanges.length > 1 ? 's' : ''} pending review
                </span>
              </div>
              <div className="flex gap-2">
                {pendingChanges.map(c => (
                  <div key={c.id} className="flex gap-1">
                    <button
                      onClick={() => acceptChange(c.id)}
                      className="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 transition-all"
                    >
                      ✓ Accept
                    </button>
                    <button
                      onClick={() => rejectChange(c.id)}
                      className="text-[10px] px-2 py-0.5 rounded bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-all"
                    >
                      ✗ Reject
                    </button>
                  </div>
                )).slice(0, 2)}
                {pendingChanges.length > 2 && (
                  <span className="text-[10px] text-gray-500">+{pendingChanges.length - 2} more in chat</span>
                )}
              </div>
            </div>
          )}

          {/* Editor */}
          <div className="flex-1 glass rounded-xl border border-white/5 overflow-hidden flex flex-col resume-editor-pane">
            <div className="flex items-center gap-4 px-4 py-1.5 border-b border-white/5 bg-surface-100/50 flex-shrink-0 no-print">
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => setActiveSubTab('editor')}
                  className={`px-3 py-1 rounded-lg text-xs font-semibold transition-all duration-150 ${
                    activeSubTab === 'editor'
                      ? 'bg-brand-600/20 text-brand-300 border border-brand-500/30'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  ✏️ Visual Editor
                </button>
                <button
                  onClick={() => setActiveSubTab('pdf')}
                  className={`px-3 py-1 rounded-lg text-xs font-semibold transition-all duration-150 ${
                    activeSubTab === 'pdf'
                      ? 'bg-brand-600/20 text-brand-300 border border-brand-500/30'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  📄 Original PDF
                </button>
              </div>
              <span className="ml-auto text-[10px] text-gray-600">
                {editorContent.length.toLocaleString()} chars
              </span>
            </div>

            {/* Printable area */}
            <div className={`flex-1 overflow-auto print-area bg-[#141416] p-8 ${activeSubTab === 'pdf' ? 'flex flex-col' : 'items-start'}`}>
              {activeSubTab === 'editor' ? (
                <div 
                  id="resume-editor-canvas"
                  contentEditable
                  suppressContentEditableWarning
                  onBlur={(e) => {
                    const newText = e.currentTarget.innerText;
                    if (newText !== editorContent) {
                      updateContent(newText);
                    }
                  }}
                  dangerouslySetInnerHTML={{ __html: formatResumeTextToHTML(editorContent) }}
                  className="editor-textarea bg-white text-slate-800 shadow-2xl rounded-sm mx-auto flex-shrink-0"
                  style={{
                    width: '8.5in',
                    minHeight: '11in',
                    padding: '1.0in 0.8in 1.0in 0.8in',
                    textAlign: 'left',
                    outline: 'none',
                    boxSizing: 'border-box'
                  }}
                  spellCheck
                />
              ) : (
                <div className="w-full flex-1 flex flex-col items-center justify-center p-4">
                  {hasPDF ? (
                    <iframe
                      src="/api/v1/resume/active/pdf"
                      className="w-full flex-1 border border-white/10 rounded-lg shadow-2xl bg-white"
                      style={{
                        minHeight: '600px',
                      }}
                      title="Original PDF Resume"
                    />
                  ) : (
                    <div className="text-center p-8 max-w-sm glass rounded-xl border border-white/5">
                      <FileText className="w-12 h-12 text-gray-600 mx-auto mb-3" />
                      <h4 className="text-sm font-semibold text-white mb-2">No PDF Data Available</h4>
                      <p className="text-xs text-gray-400 leading-relaxed mb-4">
                        We have upgraded the backend to support direct PDF rendering. To view your original PDF, please re-upload your resume.
                      </p>
                      <button
                        onClick={() => uploadInputRef.current?.click()}
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

          {/* Skill chips after save */}
          {lastSavedSkills.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1 no-print">
              <span className="text-[10px] text-gray-600 self-center mr-1">Detected:</span>
              {lastSavedSkills.slice(0, 8).map(s => (
                <span key={s} className="skill-pill-green text-[10px] py-0.5">{s}</span>
              ))}
              {lastSavedSkills.length > 8 && (
                <span className="text-[10px] text-gray-600 self-center">+{lastSavedSkills.length - 8} more</span>
              )}
            </div>
          )}
        </div>

        {/* ── Chat Pane ────────────────────────────────────────────────────── */}
        {chatVisible && (
          <div className="flex-1 min-w-0 max-w-md glass rounded-xl border border-white/5 overflow-hidden flex flex-col no-print">
            <ChatPanel
              messages={chatMessages}
              loading={isChatLoading}
              trackedChanges={trackedChanges}
              onSend={sendMessage}
              onAccept={acceptChange}
              onReject={rejectChange}
              onAnswerGap={answerGapQuestion}
              jobTitle={selectedJob?.title}
            />
          </div>
        )}
      </div>

      {/* Applied Dialog */}
      <AppliedDialog
        isOpen={showAppliedDialog}
        onClose={() => setShowAppliedDialog(false)}
        currentText={editorContent}
        selectedJob={selectedJob}
        onSaveVersion={saveAsApplied}
        onResumeUploaded={handleResumeUploaded}
      />
    </>
  );
};

export default ResumeEditor;
