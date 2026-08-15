import React, { useEffect, useState, useRef, useCallback } from 'react';
import {
  Download, Save, Upload, Clock, PanelRight, PanelRightClose,
  FileText, AlertCircle, Loader2, CheckCircle2, GitBranch,
  Sparkles, Cpu, Zap, ThumbsUp, ThumbsDown, CornerDownLeft, Check, X, RotateCcw,
  Code, Copy, ExternalLink, FileCode, Wand2
} from 'lucide-react';
import { Job, ResumeVersion, TrackedChange } from '../../types';
import { getResumeFullText, getActiveResume, analyzeJob, getResumeVersions, getVersionText, uploadResume, convertResumeToTemplate } from '../../services/api';
import { useResumeEditor } from '../../hooks/useResumeEditor';
import ChatPanel from './ChatPanel';
import ATSScoreBar from './ATSScoreBar';
import AppliedDialog from './AppliedDialog';
import ResumeUploader from './ResumeUploader';

// Helper to parse plain text resume into highly styled, beautiful HTML representation matching user's template
// Build a complete, self-contained HTML doc for high-fidelity PDF printing
const generatePrintHTML = (text: string, fitToPage: boolean = false, autoHighlight: boolean = false): string => {
  if (!text) return '';

  const parsed = parseResumeStructure(text);

  let bodyHtml = `
    <header>
        <h1>${escapeHTML(parsed.name || 'Venugopal Hegde')}</h1>
        <div class="contact-info">
            ${parsed.title ? `<strong>${escapeHTML(parsed.title)}</strong><br>` : ''}
            ${parsed.contactItems.map(item => `<span>${escapeHTML(item)}</span>`).join(' | ')}
        </div>
    </header>
  `;

  // Render sections in exact required order: Summary -> Work Experiences -> Educations -> Skills -> Any remaining
  if (parsed.summary) {
    bodyHtml += `
    <section>
        <h2>PROFESSIONAL SUMMARY</h2>
        <p>${formatTextWithHighlights(parsed.summary, autoHighlight)}</p>
    </section>`;
  }

  if (parsed.workExperience.length > 0) {
    bodyHtml += `
    <section>
        <h2>WORK EXPERIENCES</h2>
        ${parsed.workExperience.map(job => `
            <div class="job">
                <div class="job-header">
                    <span class="job-title">${escapeHTML(job.title)}</span>
                    <span class="job-date">${escapeHTML(job.date)}</span>
                </div>
                ${(job.company || job.location) ? `
                <div class="company-info">
                    <span>${escapeHTML(job.company)}</span>
                    <span>${escapeHTML(job.location)}</span>
                </div>` : ''}
                <ul>
                    ${job.bullets.map(b => `<li>${formatTextWithHighlights(b, autoHighlight)}</li>`).join('')}
                </ul>
                ${job.techStack ? `<div class="tech-stack">Technologies / Skills Used : ${formatTextWithHighlights(job.techStack, autoHighlight)}</div>` : ''}
            </div>
        `).join('')}
    </section>`;
  }

  if (parsed.education.length > 0) {
    bodyHtml += `
    <section>
        <h2>EDUCATIONS</h2>
        ${parsed.education.map(edu => `
            <div class="education-block">
                <div class="job-header">
                    <span class="job-title">${escapeHTML(edu.institution)}</span>
                    <span class="job-date">${escapeHTML(edu.date)}</span>
                </div>
                <div>${formatTextWithHighlights(edu.degree, autoHighlight)}</div>
            </div>
        `).join('')}
    </section>`;
  }

  if (parsed.skills.length > 0) {
    bodyHtml += `
    <section>
        <h2>SKILLS</h2>
        <ul class="skills-list">
            ${parsed.skills.map(skill => `
                <li><span class="skill-category">${escapeHTML(skill.category)} :</span> ${formatTextWithHighlights(skill.items, autoHighlight)}</li>
            `).join('')}
        </ul>
    </section>`;
  }

  // Render any additional custom sections (Certifications, Projects, etc.)
  parsed.customSections.forEach(sec => {
    bodyHtml += `
    <section>
        <h2>${escapeHTML(sec.title)}</h2>
        ${sec.content.map(c => `<p>${formatTextWithHighlights(c, autoHighlight)}</p>`).join('')}
    </section>`;
  });

  return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${escapeHTML(parsed.name || 'Resume')}</title>
    <style>
        @page {
            size: letter;
            margin: ${fitToPage ? '20px 25px' : '40px 30px'};
        }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: ${fitToPage ? '1.35' : '1.5'};
            color: #333;
            max-width: 850px;
            margin: 0 auto;
            padding: ${fitToPage ? '15px 15px' : '30px 20px'};
            font-size: ${fitToPage ? '9.5pt' : '10.5pt'};
            background: #fff;
        }
        header {
            text-align: left;
            margin-bottom: ${fitToPage ? '10px' : '18px'};
            border-bottom: 2px solid #2c3e50;
            padding-bottom: ${fitToPage ? '6px' : '12px'};
        }
        h1 {
            font-size: ${fitToPage ? '1.8em' : '2.2em'};
            color: #2c3e50;
            margin: 0 0 4px 0;
            font-weight: 800;
            letter-spacing: 0.5px;
            text-align: left;
        }
        .contact-info {
            font-size: ${fitToPage ? '0.88em' : '0.95em'};
            color: #555;
            text-align: left;
        }
        .contact-info span {
            margin: 0 6px;
        }
        h2 {
            color: #2c3e50;
            border-bottom: 1px solid #ccc;
            padding-bottom: 3px;
            margin-top: ${fitToPage ? '8px' : '18px'};
            margin-bottom: ${fitToPage ? '4px' : '10px'};
            font-size: ${fitToPage ? '1.05em' : '1.2em'};
            text-transform: uppercase;
            text-align: left;
        }
        .job {
            margin-bottom: ${fitToPage ? '8px' : '16px'};
        }
        .job-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 3px;
        }
        .job-title {
            font-size: 1.05em;
            font-weight: bold;
            color: #34495e;
        }
        .job-date {
            font-weight: bold;
            color: #7f8c8d;
            font-size: 0.95em;
        }
        .company-info {
            display: flex;
            justify-content: space-between;
            font-style: italic;
            color: #555;
            margin-bottom: 6px;
            font-size: 0.95em;
        }
        ul {
            margin-top: 0;
            margin-bottom: 4px;
            padding-left: 18px;
        }
        li {
            margin-bottom: ${fitToPage ? '2px' : '5px'};
            text-align: left;
        }
        .tech-stack {
            font-weight: bold;
            font-size: 0.88em;
            color: #2c3e50;
            margin-top: 4px;
        }
        .skills-list {
            list-style-type: none;
            padding: 0;
        }
        .skills-list li {
            margin-bottom: ${fitToPage ? '4px' : '8px'};
        }
        .skill-category {
            font-weight: bold;
            color: #2c3e50;
            width: 210px;
            display: inline-block;
        }
        .education-block {
            margin-bottom: ${fitToPage ? '8px' : '12px'};
        }
        .kw-highlight {
            font-weight: 700;
            color: #1e3a8a;
            background-color: rgba(59, 130, 246, 0.12);
            padding: 0px 3px;
            border-radius: 3px;
        }
        @media print {
            body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
        }
    </style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
};

// Helper utilities for parser & HTML formatting
function escapeHTML(str: string): string {
  return str.replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}

const IMPORTANT_KEYWORDS = [
  'Go', 'Golang', 'Kubernetes', 'Docker', 'Google Pub/Sub', 'Pub/Sub', 'Redis', 'PostgreSQL',
  'DynamoDB', 'MongoDB', 'NATS', 'Helm', 'AWS', 'Azure', 'GCP', 'gRPC', 'Python', 'TypeScript',
  'SQL', 'Shell Scripting', 'Platform Engineering', 'Test-Driven Development', 'TDD', 'System Design',
  'Agile', 'Scrum', 'Agile/Scrum', 'Remote Collaboration', 'CI/CD Automation', 'CI/CD', 'REST API',
  'WebSocket', 'Mocha', 'SLB OSDU Data Platform', 'OSDU'
];

function formatTextWithHighlights(text: string, highlight: boolean, customKeywords?: string[]): string {
  let escaped = escapeHTML(text);
  if (!highlight) return escaped;

  const keywordsToUse = (customKeywords && customKeywords.length > 0) 
    ? Array.from(new Set([...customKeywords, ...IMPORTANT_KEYWORDS])) 
    : IMPORTANT_KEYWORDS;

  // Highlight key technologies & terms
  keywordsToUse.forEach(kw => {
    if (!kw || kw.trim().length < 2) return;
    const escKw = escapeHTML(kw.trim());
    const rx = new RegExp(`\\b(${escKw.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&')})\\b`, 'gi');
    escaped = escaped.replace(rx, `<span class="kw-highlight">$1</span>`);
  });
  return escaped;
}

interface ParsedJob {
  title: string;
  date: string;
  company: string;
  location: string;
  bullets: string[];
  techStack?: string;
}

interface ParsedEdu {
  institution: string;
  date: string;
  degree: string;
}

interface ParsedSkill {
  category: string;
  items: string;
}

interface ParsedCustomSection {
  title: string;
  content: string[];
}

interface ParsedResume {
  name: string;
  title: string;
  contactItems: string[];
  summary: string;
  skills: ParsedSkill[];
  workExperience: ParsedJob[];
  education: ParsedEdu[];
  customSections: ParsedCustomSection[];
}

function convertStructuredToText(sr: any): string {
  if (!sr) return '';
  const lines: string[] = [];

  if (sr.name) lines.push(sr.name.trim());
  if (sr.title) lines.push(sr.title.trim());
  if (sr.contact_items && sr.contact_items.length > 0) {
    lines.push(sr.contact_items.join(' | '));
  } else if (sr.contactItems && sr.contactItems.length > 0) {
    lines.push(sr.contactItems.join(' | '));
  }

  lines.push('');

  if (sr.summary) {
    lines.push('PROFESSIONAL SUMMARY');
    lines.push(sr.summary.trim());
    lines.push('');
  }

  const work = sr.work_experience || sr.workExperience || [];
  if (work.length > 0) {
    lines.push('WORK EXPERIENCES');
    work.forEach((job: any) => {
      const titleLine = `${job.title || ''} ${job.date ? job.date : ''}`.trim();
      if (titleLine) lines.push(titleLine);
      if (job.company || job.location) {
        const compLine = [job.company, job.location].filter(Boolean).join(' | ');
        lines.push(compLine);
      }
      if (job.bullets && job.bullets.length > 0) {
        job.bullets.forEach((b: string) => {
          if (b && b.trim()) lines.push(`• ${b.trim()}`);
        });
      }
      if (job.tech_stack || job.techStack) {
        lines.push(`Technologies / Skills Used : ${job.tech_stack || job.techStack}`);
      }
      lines.push('');
    });
  }

  const edu = sr.education || [];
  if (edu.length > 0) {
    lines.push('EDUCATIONS');
    edu.forEach((e: any) => {
      const instLine = `${e.institution || ''} ${e.date ? e.date : ''}`.trim();
      if (instLine) lines.push(instLine);
      if (e.degree) lines.push(e.degree.trim());
      lines.push('');
    });
  }

  const skills = sr.skills || [];
  if (skills.length > 0) {
    lines.push('SKILLS');
    skills.forEach((s: any) => {
      if (s.category && s.items) {
        lines.push(`${s.category.trim()} : ${s.items.trim()}`);
      }
    });
    lines.push('');
  }

  return lines.join('\n').trim();
}

function parseResumeStructure(text: string): ParsedResume {
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean);
  const result: ParsedResume = {
    name: '',
    title: '',
    contactItems: [],
    summary: '',
    skills: [],
    workExperience: [],
    education: [],
    customSections: []
  };

  if (lines.length === 0) return result;

  // Header extraction
  result.name = lines[0].replace(/\[cite:\s*\d+\]/g, '');
  let idx = 1;
  while (idx < lines.length) {
    const line = lines[idx].replace(/\[cite:\s*\d+\]/g, '');
    const isSectionHeader = /^(PROFESSIONAL SUMMARY|SUMMARY|WORK EXPERIENCES|EXPERIENCE|SKILLS|EDUCATIONS|EDUCATION|PROJECTS|CERTIFICATIONS)$/i.test(line);
    if (isSectionHeader) break;

    if (!result.title && !/[@|+|\d{5,}]/.test(line)) {
      result.title = line;
    } else {
      const parts = line.split(/[|•]/).map(p => p.trim()).filter(Boolean);
      parts.forEach(p => result.contactItems.push(p));
    }
    idx++;
  }

  // Parse Sections
  let currentSection = '';
  let currentJob: ParsedJob | null = null;
  let currentEdu: ParsedEdu | null = null;

  for (; idx < lines.length; idx++) {
    const rawLine = lines[idx];
    const line = rawLine.replace(/\[cite:\s*\d+\]/g, '').trim();
    if (!line) continue;

    const isSectionHeader = /^(PROFESSIONAL SUMMARY|SUMMARY|WORK EXPERIENCES|WORK EXPERIENCE|EXPERIENCE|SKILLS|TECHNICAL SKILLS|EDUCATIONS|EDUCATION|PROJECTS|CERTIFICATIONS)$/i.test(line)
      || (line.length < 40 && line === line.toUpperCase() && !line.includes('|') && !line.includes('@') && !line.includes('+') && line.length > 3);

    if (isSectionHeader) {
      if (currentJob) { result.workExperience.push(currentJob); currentJob = null; }
      if (currentEdu) { result.education.push(currentEdu); currentEdu = null; }

      if (/SUMMARY/i.test(line)) currentSection = 'SUMMARY';
      else if (/SKILL/i.test(line)) currentSection = 'SKILLS';
      else if (/WORK|EXPERIENCE/i.test(line)) currentSection = 'WORK';
      else if (/EDU/i.test(line)) currentSection = 'EDUCATION';
      else {
        currentSection = line;
        result.customSections.push({ title: line, content: [] });
      }
      continue;
    }

    if (currentSection === 'SUMMARY') {
      result.summary = result.summary ? result.summary + ' ' + line : line;
    } else if (currentSection === 'SKILLS') {
      if (line.includes(':')) {
        const [cat, items] = line.split(':', 2);
        result.skills.push({ category: cat.trim(), items: items.trim() });
      } else {
        const isBullet = /^[•\-*▪◦]/.test(line);
        const txt = line.replace(/^[•\-*▪◦\s]+/, '').trim();
        result.skills.push({ category: 'Key Skills', items: txt });
      }
    } else if (currentSection === 'WORK') {
      const dateRx = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\u2013\u2014]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))\s*$/i;
      const dm = line.match(dateRx);

      if (dm && line.length < 130) {
        if (currentJob) { result.workExperience.push(currentJob); }
        const date = dm[1];
        const title = line.replace(dateRx, '').trim().replace(/[\s|–—-]+$/, '').trim();
        currentJob = { title, date, company: '', location: '', bullets: [] };
      } else if (line.toLowerCase().startsWith('technologies') || line.toLowerCase().startsWith('technologies / skills used')) {
        if (currentJob) {
          const tech = line.replace(/^technologies\s*\/\s*skills\s*used\s*:\s*/i, '').replace(/^technologies\s*used\s*:\s*/i, '').trim();
          currentJob.techStack = tech;
        }
      } else if ((line.includes('|') || line.includes(' - ') || line.includes('–')) && currentJob && !currentJob.company) {
        const parts = line.split(/[|–-]/).map(p => p.trim()).filter(Boolean);
        if (parts.length >= 2) {
          currentJob.company = parts[0];
          currentJob.location = parts.slice(1).join(' - ');
        } else {
          currentJob.company = line;
        }
      } else if (currentJob) {
        const isBullet = /^[•\-*▪◦]/.test(line);
        const txt = line.replace(/^[•\-*▪◦\s]+/, '').trim();
        currentJob.bullets.push(txt);
      }
    } else if (currentSection === 'EDUCATION') {
      const dateRx = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\u2013\u2014]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))\s*$/i;
      const dm = line.match(dateRx);
      if (dm) {
        if (currentEdu) { result.education.push(currentEdu); }
        const date = dm[1];
        const inst = line.replace(dateRx, '').trim().replace(/[\s|–—-]+$/, '').trim();
        currentEdu = { institution: inst, date, degree: '' };
      } else if (currentEdu) {
        currentEdu.degree = currentEdu.degree ? currentEdu.degree + ' ' + line : line;
      } else {
        result.education.push({ institution: line, date: '', degree: '' });
      }
    } else {
      const cust = result.customSections.find(s => s.title === currentSection);
      if (cust) cust.content.push(line);
    }
  }

  if (currentJob) result.workExperience.push(currentJob);
  if (currentEdu) result.education.push(currentEdu);

  return result;
}

const renderEditorCanvasHTML = (text: string, fitToPage: boolean = false, autoHighlight: boolean = false): string => {
  if (!text) return '';
  const parsed = parseResumeStructure(text);

  let bodyHtml = `
    <header>
        <h1>${escapeHTML(parsed.name || 'Venugopal Hegde')}</h1>
        <div class="contact-info">
            ${parsed.title ? `<strong>${escapeHTML(parsed.title)}</strong><br>` : ''}
            ${parsed.contactItems.map(item => `<span>${escapeHTML(item)}</span>`).join(' | ')}
        </div>
    </header>
  `;

  if (parsed.summary) {
    bodyHtml += `
    <section>
        <h2>PROFESSIONAL SUMMARY</h2>
        <p>${formatTextWithHighlights(parsed.summary, autoHighlight)}</p>
    </section>`;
  }

  if (parsed.workExperience.length > 0) {
    bodyHtml += `
    <section>
        <h2>WORK EXPERIENCES</h2>
        ${parsed.workExperience.map(job => `
            <div class="job">
                <div class="job-header">
                    <span class="job-title">${escapeHTML(job.title)}</span>
                    <span class="job-date">${escapeHTML(job.date)}</span>
                </div>
                ${(job.company || job.location) ? `
                <div class="company-info">
                    <span>${escapeHTML(job.company)}</span>
                    <span>${escapeHTML(job.location)}</span>
                </div>` : ''}
                <ul>
                    ${job.bullets.map(b => `<li>${formatTextWithHighlights(b, autoHighlight)}</li>`).join('')}
                </ul>
                ${job.techStack ? `<div class="tech-stack">Technologies / Skills Used : ${formatTextWithHighlights(job.techStack, autoHighlight)}</div>` : ''}
            </div>
        `).join('')}
    </section>`;
  }

  if (parsed.education.length > 0) {
    bodyHtml += `
    <section>
        <h2>EDUCATIONS</h2>
        ${parsed.education.map(edu => `
            <div class="education-block">
                <div class="job-header">
                    <span class="job-title">${escapeHTML(edu.institution)}</span>
                    <span class="job-date">${escapeHTML(edu.date)}</span>
                </div>
                <div>${formatTextWithHighlights(edu.degree, autoHighlight)}</div>
            </div>
        `).join('')}
    </section>`;
  }

  if (parsed.skills.length > 0) {
    bodyHtml += `
    <section>
        <h2>SKILLS</h2>
        <ul class="skills-list">
            ${parsed.skills.map(skill => `
                <li><span class="skill-category">${escapeHTML(skill.category)} :</span> ${formatTextWithHighlights(skill.items, autoHighlight)}</li>
            `).join('')}
        </ul>
    </section>`;
  }

  parsed.customSections.forEach(sec => {
    bodyHtml += `
    <section>
        <h2>${escapeHTML(sec.title)}</h2>
        ${sec.content.map(c => `<p>${formatTextWithHighlights(c, autoHighlight)}</p>`).join('')}
    </section>`;
  });

  return bodyHtml;
};

const formatResumeTextToHTML = (text: string, autoHighlight: boolean = false, fitToPage: boolean = false): string => {
  if (!text) return '';
  return renderEditorCanvasHTML(text, fitToPage, autoHighlight);
};

interface MatchResult {
  start: number;
  end: number;
}

const expandToWordBoundaries = (text: string, start: number, end: number): MatchResult => {
  let newStart = start;
  let newEnd = end;
  const isWordChar = (char: string) => /[a-zA-Z0-9_]/.test(char);

  if (start > 0 && isWordChar(text[start]) && isWordChar(text[start - 1])) {
    while (newStart > 0 && isWordChar(text[newStart - 1])) {
      newStart--;
    }
  }

  if (end < text.length && isWordChar(text[end - 1]) && isWordChar(text[end])) {
    while (newEnd < text.length && isWordChar(text[newEnd])) {
      newEnd++;
    }
  }

  return { start: newStart, end: newEnd };
};

const findFlexibleMatch = (text: string, pattern: string): MatchResult | null => {
  const normalize = (str: string) => str.replace(/[\s\u200b\u200c\u200d\ufeff]+/g, ' ').trim().toLowerCase();
  
  const cleanText = text.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
  const normPattern = normalize(pattern);
  if (!normPattern) return null;

  const stripPrefix = (str: string) => str.replace(/^[•\-\*\s]+/, '').trim();
  const normPatternClean = stripPrefix(normPattern);
  if (!normPatternClean) return null;

  const words = normPatternClean.split(' ').filter(w => w !== '').map(w => w.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&'));
  if (words.length === 0) return null;

  try {
    const regexStr = '[•\\-\\*\\s]*' + words.join('[\\s\\r\\n\\W]*');
    const regex = new RegExp(regexStr, 'i');
    const match = cleanText.match(regex);
    if (match && match.index !== undefined) {
      return expandToWordBoundaries(cleanText, match.index, match.index + match[0].length);
    }
  } catch (e) {
    // Ignore regex errors
  }

  const cleanPattern = pattern.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
  const idx = cleanText.indexOf(cleanPattern);
  if (idx !== -1) {
    return expandToWordBoundaries(cleanText, idx, idx + cleanPattern.length);
  }

  const cleanPatternNoPunct = cleanPattern.trim().replace(/[;,.:\s]+$/, '');
  const idx2 = cleanText.indexOf(cleanPatternNoPunct);
  if (idx2 !== -1) {
    return expandToWordBoundaries(cleanText, idx2, idx2 + cleanPatternNoPunct.length);
  }

  return null;
};

interface AppliedMatch {
  change: TrackedChange;
  start: number;
  end: number;
}

const applyPendingChangesToText = (text: string, pendingChanges: TrackedChange[]): string => {
  const matches: AppliedMatch[] = [];

  // 1. Find all matches in the original unmodified text
  pendingChanges.forEach(change => {
    if (change.status !== 'pending') return;
    const match = findFlexibleMatch(text, change.original);
    if (match) {
      matches.push({ change, start: match.start, end: match.end });
    }
  });

  // 2. Filter out overlapping matches
  // Sort matches by start index ascending for overlap check
  matches.sort((a, b) => a.start - b.start);
  const nonOverlapping: AppliedMatch[] = [];
  let lastEnd = -1;

  for (const m of matches) {
    if (m.start >= lastEnd) {
      nonOverlapping.push(m);
      lastEnd = m.end;
    } else {
      console.warn(`[AI Editor] Discarding overlapping render for: "${m.change.original}"`);
    }
  }

  // 3. Sort non-overlapping matches descending (back-to-front)
  nonOverlapping.sort((a, b) => b.start - a.start);

  // 4. Apply replacements from back to front
  let result = text;
  for (const m of nonOverlapping) {
    const matchedText = result.slice(m.start, m.end);
    if (matchedText.includes('<del') || matchedText.includes('<ins') || matchedText.includes('class=')) {
      continue;
    }
    const delTag = `<del class="bg-rose-500/10 text-rose-600 line-through decoration-rose-500 cursor-pointer px-0.5 rounded select-all font-medium inline" data-edit-id="${m.change.id}">${matchedText}</del>`;
    const insTag = `<ins class="bg-emerald-500/10 text-emerald-600 no-underline cursor-pointer border-b border-dashed border-emerald-500 px-0.5 rounded font-medium inline" data-edit-id="${m.change.id}">${m.change.replacement}</ins>`;
    result = result.slice(0, m.start) + delTag + insTag + result.slice(m.end);
  }

  return result;
};

const getCleanTextFromDOM = (node: Node): string => {
  if (node.nodeType === Node.TEXT_NODE) {
    return node.nodeValue || '';
  }
  if (node.nodeType === Node.ELEMENT_NODE) {
    const el = node as HTMLElement;
    if (el.tagName === 'DEL') {
      return '';
    }
    if (el.tagName === 'BR') {
      return '\n';
    }
    let text = '';
    for (let i = 0; i < el.childNodes.length; i++) {
      text += getCleanTextFromDOM(el.childNodes[i]);
    }
    const isBlock = ['P', 'DIV', 'H1', 'H2', 'H3', 'LI'].includes(el.tagName);
    if (isBlock && !text.endsWith('\n')) {
      text += '\n';
    }
    return text;
  }
  return '';
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
    revertContent,
    setChatMessages,
    applyFullResume,
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
  const [selectedEditId, setSelectedEditId] = useState<string | null>(null);
  const [cardPosition, setCardPosition] = useState<{ x: number, y: number } | null>(null);
  const [refineInput, setRefineInput] = useState('');
  const [activeModel, setActiveModel] = useState<string>('');
  

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
    // Clear chat history when the target job changes to isolate sessions
    setChatMessages([]);
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

  const runATS = useCallback(async (force = true) => {
    if (!selectedJob) return;
    setAtsLoading(true);
    try {
      const result = await analyzeJob(selectedJob.id, undefined, force);
      setPrevAtsScore(atsScore);
      setAtsScore(result.ats_score);
    } catch {
      // silently ignore
    } finally {
      setAtsLoading(false);
    }
  }, [selectedJob, atsScore]);

  const [fitToSinglePage, setFitToSinglePage] = useState(false);
  const [highlightKeywords, setHighlightKeywords] = useState(true);

  // Export Modal state & handlers
  const [showExportModal, setShowExportModal] = useState(false);
  const [exportingAI, setExportingAI] = useState(false);
  const [isFormattingAI, setIsFormattingAI] = useState(false);
  const [exportHTML, setExportHTML] = useState('');
  const [exportTab, setExportTab] = useState<'preview' | 'code'>('preview');
  const [copySuccess, setCopySuccess] = useState(false);

  const handleOpenExportModal = () => {
    const html = generatePrintHTML(editorContent, fitToSinglePage, highlightKeywords);
    setExportHTML(html);
    setShowExportModal(true);
  };

  const handleToggleFitToSinglePage = async (checked: boolean) => {
    setFitToSinglePage(checked);
    if (checked) {
      setIsFormattingAI(true);
      try {
        const res = await convertResumeToTemplate(editorContent, activeModel, true);
        if (res) {
          if (res.parsed) {
            const textToUse = convertStructuredToText(res.parsed);
            applyFullResume(textToUse);
            await saveContent(textToUse);
          }
          if (res.html) {
            setExportHTML(res.html);
          } else {
            setExportHTML(generatePrintHTML(editorContent, true, highlightKeywords));
          }
          setSaveMessage('AI Optimized for 1 Single Page!');
          if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
          saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
          if (selectedJob) runATS();
        }
      } catch (err) {
        console.error('Failed 1-page fit optimization:', err);
        setExportHTML(generatePrintHTML(editorContent, true, highlightKeywords));
      } finally {
        setIsFormattingAI(false);
      }
    } else {
      setExportHTML(generatePrintHTML(editorContent, false, highlightKeywords));
    }
  };

  const handleFormatPDFWithAI = async () => {
    setIsFormattingAI(true);
    try {
      const res = await convertResumeToTemplate(editorContent, activeModel, fitToSinglePage);
      if (res) {
        if (res.parsed) {
          const textToUse = convertStructuredToText(res.parsed);
          applyFullResume(textToUse);
          await saveContent(textToUse);
        }
        if (res.html) {
          setExportHTML(res.html);
        }
        setShowExportModal(true);
        setSaveMessage('PDF Formatted with AI!');
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
        if (selectedJob) runATS();
      }
    } catch (err) {
      console.error('Failed to format PDF with AI:', err);
      handleOpenExportModal();
    } finally {
      setIsFormattingAI(false);
    }
  };

  const handleConvertWithAI = async () => {
    setExportingAI(true);
    try {
      const res = await convertResumeToTemplate(editorContent, activeModel, fitToSinglePage);
      if (res) {
        if (res.parsed) {
          const textToUse = convertStructuredToText(res.parsed);
          applyFullResume(textToUse);
          await saveContent(textToUse);
        }
        if (res.html) {
          setExportHTML(res.html);
        }
        if (selectedJob) runATS();
      }
    } catch (err) {
      console.error('Failed to convert resume via AI:', err);
      setExportHTML(generatePrintHTML(editorContent, fitToSinglePage, highlightKeywords));
    } finally {
      setExportingAI(false);
    }
  };

  const handleDownloadHTML = () => {
    const htmlToSave = exportHTML || generatePrintHTML(editorContent, fitToSinglePage, highlightKeywords);
    const blob = new Blob([htmlToSave], { type: 'text/html;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'Venugopal_Hegde_Resume.html';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleCopyHTML = () => {
    const htmlToCopy = exportHTML || generatePrintHTML(editorContent, fitToSinglePage, highlightKeywords);
    navigator.clipboard.writeText(htmlToCopy);
    setCopySuccess(true);
    setTimeout(() => setCopySuccess(false), 2000);
  };

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

  const handleRevert = useCallback(async () => {
    if (!window.confirm('Are you sure you want to revert all changes? This will restore the original resume and convert it to the standard ATS template.')) {
      return;
    }
    const result = await revertContent();
    if (result) {
      setChatMessages([]); // Reset chat history on revert to start a clean session
      let textToUse = result.text;
      if (result.parsed) {
        textToUse = convertStructuredToText(result.parsed);
      }
      applyFullResume(textToUse);
      await saveContent(textToUse);

      if (result.html) {
        setExportHTML(result.html);
      } else {
        const html = generatePrintHTML(textToUse, fitToSinglePage, highlightKeywords);
        setExportHTML(html);
      }

      setSaveMessage('Reverted & Formatted to Template!');
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
      // Re-run ATS after reverting
      if (selectedJob) runATS();
    }
  }, [revertContent, selectedJob, runATS, setChatMessages, fitToSinglePage, highlightKeywords, applyFullResume, saveContent]);

  const handleApplyFullResume = useCallback(async (text: string) => {
    applyFullResume(text);
    await saveContent(text);
    if (selectedJob) runATS();
  }, [applyFullResume, saveContent, selectedJob, runATS]);

  const handleDownloadPDF = () => {
    const resumeHTML = generatePrintHTML(editorContent, fitToSinglePage, highlightKeywords);
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
      {/* Sleek Top Command Bar */}
      <div className="glass rounded-xl border border-white/10 p-3.5 mb-4 no-print space-y-3 shadow-xl">
        {/* Row 1: ATS Score + Core Controls */}
        <div className="flex flex-wrap items-center justify-between gap-3">
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

          <div className="flex items-center gap-2">
            {/* View Switcher Tabs */}
            <div className="flex items-center bg-surface-300 rounded-lg p-1 border border-white/5">
              <button
                onClick={() => setActiveSubTab('editor')}
                className={`px-3 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all ${
                  activeSubTab === 'editor'
                    ? 'bg-indigo-600 text-white shadow-md'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                ✏️ Visual Canvas
              </button>
              <button
                onClick={() => setActiveSubTab('pdf')}
                className={`px-3 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all ${
                  activeSubTab === 'pdf'
                    ? 'bg-indigo-600 text-white shadow-md'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                📄 Original PDF
              </button>
            </div>

            {/* Chat Panel Toggle */}
            <button
              id="btn-toggle-chat"
              onClick={() => setChatVisible(!chatVisible)}
              className={`p-2 rounded-lg text-xs font-medium border flex items-center gap-1.5 transition-all ${
                chatVisible
                  ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
                  : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
              }`}
              title={chatVisible ? 'Hide AI Copilot' : 'Show AI Copilot'}
            >
              {chatVisible ? <PanelRightClose className="w-4 h-4" /> : <PanelRight className="w-4 h-4" />}
              <span className="hidden sm:inline">Copilot</span>
            </button>
          </div>
        </div>

        {/* Row 2: Density Options & Quick AI Actions */}
        <div className="flex flex-wrap items-center justify-between gap-2.5 pt-2 border-t border-white/5">
          {/* Left Options */}
          <div className="flex items-center gap-2 flex-wrap">
            <input
              type="file"
              ref={uploadInputRef}
              style={{ display: 'none' }}
              accept=".pdf,.txt"
              onChange={handleDirectUpload}
            />

            {/* Fit to 1 Page Button */}
            <button
              id="btn-fit-single-page"
              onClick={() => handleToggleFitToSinglePage(!fitToSinglePage)}
              disabled={isFormattingAI}
              className={`text-xs px-3 py-1.5 rounded-lg flex items-center gap-1.5 font-medium transition-all border ${
                fitToSinglePage
                  ? 'bg-indigo-600 text-white border-indigo-400 shadow-md shadow-indigo-600/30'
                  : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
              }`}
              title="Ask AI to condense and fit resume content strictly onto 1 single page"
            >
              {isFormattingAI ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
              ) : (
                <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
              )}
              Fit to 1 Page {fitToSinglePage ? '✓' : ''}
            </button>

            {/* Keyword Highlight Toggle */}
            <label className="flex items-center gap-1.5 cursor-pointer px-3 py-1.5 rounded-lg bg-surface-200 hover:bg-surface-300 border border-white/5 transition-all text-xs font-medium text-gray-300">
              <input
                type="checkbox"
                checked={highlightKeywords}
                onChange={(e) => setHighlightKeywords(e.target.checked)}
                className="rounded accent-indigo-500 w-3.5 h-3.5 cursor-pointer"
              />
              Highlight Keywords
            </label>

            {/* History Button */}
            <button
              id="btn-version-history"
              onClick={() => { setShowVersions(!showVersions); if (!showVersions) loadVersions(); }}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 border transition-all ${
                showVersions ? 'bg-indigo-600/20 text-indigo-300 border-indigo-500/40' : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
              }`}
            >
              <GitBranch className="w-3.5 h-3.5 text-indigo-400" />
              History
            </button>

            {/* Upload New Button */}
            <button
              id="btn-direct-upload"
              onClick={() => uploadInputRef.current?.click()}
              className="px-3 py-1.5 rounded-lg bg-surface-200 hover:bg-surface-300 border border-white/5 text-xs text-gray-300 font-medium flex items-center gap-1.5 transition-all"
              title="Upload a new resume PDF/TXT"
            >
              <Upload className="w-3.5 h-3.5 text-gray-400" />
              Upload New
            </button>
          </div>

          {/* Right Action Commands */}
          <div className="flex items-center gap-2 flex-wrap ml-auto">
            {/* Status notification badge */}
            {saveMessage && (
              <span className="flex items-center gap-1 text-xs text-emerald-400 font-medium animate-fade-in mr-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> {saveMessage}
              </span>
            )}
            {isDirty && !saveMessage && (
              <span className="text-xs text-amber-400 font-medium animate-pulse mr-1">Unsaved changes</span>
            )}

            {/* Revert Button */}
            <button
              id="btn-revert-resume"
              onClick={handleRevert}
              disabled={isSaving}
              className="px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/20 text-xs font-medium flex items-center gap-1.5 disabled:opacity-40 transition-all"
              title="Revert all edits back to original PDF text and convert to standard template"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              Revert
            </button>

            {/* Save Button */}
            <button
              id="btn-save-resume"
              onClick={() => handleSave(false)}
              disabled={isSaving || !isDirty}
              className="px-3 py-1.5 rounded-lg bg-surface-200 hover:bg-surface-300 text-white border border-white/10 text-xs font-medium flex items-center gap-1.5 disabled:opacity-40 transition-all"
            >
              {isSaving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5 text-emerald-400" />}
              Save
            </button>

            {/* AI Format PDF Button */}
            <button
              id="btn-format-pdf-ai"
              onClick={handleFormatPDFWithAI}
              disabled={isFormattingAI}
              className="px-3.5 py-1.5 rounded-lg bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/30 disabled:opacity-50 transition-all"
              title="Format PDF content into standard ATS template using AI"
            >
              {isFormattingAI ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
              ) : (
                <Wand2 className="w-3.5 h-3.5 text-yellow-300" />
              )}
              Format PDF with AI
            </button>

            {/* Export ATS Template Modal Button */}
            <button
              id="btn-export-ats-template"
              onClick={handleOpenExportModal}
              className="px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/30 transition-all"
              title="Export resume using AI ATS HTML & PDF Template"
            >
              <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
              Export Template
            </button>
          </div>
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

          {/* Editor Container */}
          <div className="flex-1 glass rounded-xl border border-white/10 overflow-hidden flex flex-col resume-editor-pane shadow-2xl">
            {/* Canvas Sub-Header */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-white/5 bg-surface-100/70 flex-shrink-0 no-print">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                <span className="text-xs font-semibold text-gray-200">
                  {activeSubTab === 'editor' ? 'Visual Resume Canvas (8.5" x 11")' : 'Original Uploaded PDF Document'}
                </span>
                {fitToSinglePage && (
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 font-medium">
                    1-Page High-Density Mode
                  </span>
                )}
              </div>
              <span className="text-xs font-mono text-gray-400">
                {editorContent.length.toLocaleString()} characters
              </span>
            </div>

            {/* Printable canvas area */}
            <div 
              className={`flex-1 overflow-auto print-area bg-[#0b0f19] p-8 flex justify-center ${activeSubTab === 'pdf' ? 'flex flex-col' : 'items-start'} relative`}
              onClick={(e) => {
                // Clicking outside the canvas or edits should dismiss the card
                const target = e.target as HTMLElement;
                if (!target.closest('[data-edit-id]')) {
                  setSelectedEditId(null);
                  setCardPosition(null);
                }
              }}
            >
              {activeSubTab === 'editor' ? (
                <div 
                  id="resume-editor-canvas"
                  contentEditable
                  suppressContentEditableWarning
                  onClick={(e) => {
                    const target = e.target as HTMLElement;
                    const editEl = target.closest('[data-edit-id]');
                    if (editEl) {
                      const editId = editEl.getAttribute('data-edit-id');
                      if (editId) {
                        const rect = editEl.getBoundingClientRect();
                        const canvasEl = document.getElementById('resume-editor-canvas');
                        if (canvasEl) {
                          const parentEl = canvasEl.parentElement;
                          if (parentEl) {
                            const parentRect = parentEl.getBoundingClientRect();
                            setCardPosition({
                              x: rect.left - parentRect.left + (rect.width / 2),
                              y: rect.bottom - parentRect.top + parentEl.scrollTop + 8
                            });
                            setSelectedEditId(editId);
                            setRefineInput('');
                            return;
                          }
                        }
                      }
                    }
                  }}
                  onBlur={(e) => {
                    const newText = getCleanTextFromDOM(e.currentTarget);
                    if (newText !== editorContent) {
                      updateContent(newText);
                    }
                  }}
                  dangerouslySetInnerHTML={{ 
                    __html: formatResumeTextToHTML(applyPendingChangesToText(editorContent, pendingChanges), highlightKeywords, fitToSinglePage) 
                  }}
                  className={`editor-textarea bg-white text-slate-800 shadow-2xl rounded-sm mx-auto flex-shrink-0 ${fitToSinglePage ? 'fit-page' : ''}`}
                  style={{
                    width: '8.5in',
                    minHeight: '11in',
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

              {/* Floating Google-Docs-Style Suggestion Card */}
              {selectedEditId && cardPosition && (
                (() => {
                  const edit = pendingChanges.find(c => c.id === selectedEditId);
                  if (!edit) return null;

                  return (
                    <div
                      className="absolute z-50 w-80 glass rounded-xl border border-white/10 shadow-2xl p-4 flex flex-col gap-3 animate-fade-in bg-surface-200/95 backdrop-blur-md"
                      style={{
                        left: `${cardPosition.x}px`,
                        top: `${cardPosition.y}px`,
                        transform: 'translateX(-50%)',
                      }}
                      onClick={(e) => e.stopPropagation()} // Prevent closing when clicking card itself
                    >
                      {/* Top Action Row */}
                      <div className="flex items-center justify-between border-b border-white/5 pb-2">
                        <div className="flex items-center gap-1.5">
                          <Sparkles className="w-3.5 h-3.5 text-brand-400" />
                          <span className="text-xs font-semibold text-white">Gemini Suggestion</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => {
                              // Visual thumbs up feedback (no-op or toast)
                            }}
                            className="p-1 hover:bg-white/5 rounded text-gray-400 hover:text-white transition-all"
                            title="Helpful"
                          >
                            <ThumbsUp className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => {
                              // Visual thumbs down feedback
                            }}
                            className="p-1 hover:bg-white/5 rounded text-gray-400 hover:text-white transition-all"
                            title="Not helpful"
                          >
                            <ThumbsDown className="w-3.5 h-3.5" />
                          </button>
                          <div className="w-px h-3 bg-white/10" />
                          <button
                            onClick={() => {
                              rejectChange(selectedEditId);
                              setSelectedEditId(null);
                              setCardPosition(null);
                            }}
                            className="p-1 hover:bg-red-500/10 rounded text-red-400 hover:text-red-300 transition-all"
                            title="Reject suggestion"
                          >
                            <X className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => {
                              acceptChange(selectedEditId);
                              setSelectedEditId(null);
                              setCardPosition(null);
                            }}
                            className="p-1 hover:bg-emerald-500/10 rounded text-emerald-400 hover:text-emerald-300 transition-all"
                            title="Accept suggestion"
                          >
                            <Check className="w-4 h-4" />
                          </button>
                        </div>
                      </div>

                      {/* Explanation */}
                      <div className="space-y-2">
                        {edit.reason && (
                          <p className="text-xs text-gray-300 leading-relaxed font-normal">
                            {edit.reason}
                          </p>
                        )}
                        <div className="bg-brand-500/5 border border-brand-500/10 rounded-lg p-2 text-[11px] text-brand-300 font-mono select-all">
                          {edit.replacement}
                        </div>
                      </div>

                      {/* Refinement input */}
                      <div className="relative flex items-center mt-1 border border-white/10 rounded-lg bg-surface-300/50 focus-within:border-brand-500/50 transition-all">
                        <input
                          type="text"
                          value={refineInput}
                          onChange={(e) => setRefineInput(e.target.value)}
                          placeholder="Refine suggestion with Gemini..."
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && refineInput.trim()) {
                              // Send refinement message
                              const msg = `Regarding the suggestion to replace "${edit.original}" with "${edit.replacement}": ${refineInput.trim()}`;
                              sendMessage(msg);
                              setRefineInput('');
                              setSelectedEditId(null);
                              setCardPosition(null);
                            }
                          }}
                          className="w-full bg-transparent text-xs py-2 pl-3 pr-8 text-white outline-none placeholder:text-gray-500"
                        />
                        <button
                          onClick={() => {
                            if (refineInput.trim()) {
                              const msg = `Regarding the suggestion to replace "${edit.original}" with "${edit.replacement}": ${refineInput.trim()}`;
                              sendMessage(msg);
                              setRefineInput('');
                              setSelectedEditId(null);
                              setCardPosition(null);
                            }
                          }}
                          disabled={!refineInput.trim()}
                          className="absolute right-2 p-1 text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition-all"
                        >
                          <CornerDownLeft className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  );
                })()
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
              onSend={(text) => sendMessage(text, activeModel)}
              onAccept={acceptChange}
              onReject={rejectChange}
              onAnswerGap={answerGapQuestion}
              jobTitle={selectedJob?.title}
              activeModel={activeModel}
              onModelChange={setActiveModel}
              onApplyFullResume={handleApplyFullResume}
              onSelectEdit={(id) => {
                // Find the edit element in visual editor and trigger the card
                setTimeout(() => {
                  const editEl = document.querySelector(`[data-edit-id="${id}"]`);
                  if (editEl) {
                    editEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    const rect = editEl.getBoundingClientRect();
                    const canvasEl = document.getElementById('resume-editor-canvas');
                    if (canvasEl) {
                      const parentEl = canvasEl.parentElement;
                      if (parentEl) {
                        const parentRect = parentEl.getBoundingClientRect();
                        setCardPosition({
                          x: rect.left - parentRect.left + (rect.width / 2),
                          y: rect.bottom - parentRect.top + parentEl.scrollTop + 8
                        });
                        setSelectedEditId(id);
                        setRefineInput('');
                      }
                    }
                  }
                }, 100);
              }}
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

      {/* ── Export ATS Resume Modal ────────────────────────────────────────── */}
      {showExportModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-fade-in">
          <div className="bg-surface-100 border border-white/10 rounded-2xl w-full max-w-5xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
            {/* Modal Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-white/10 bg-surface-200/50">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-indigo-500/20 border border-indigo-500/30 flex items-center justify-center">
                  <Sparkles className="w-5 h-5 text-indigo-400" />
                </div>
                <div>
                  <h3 className="text-base font-semibold text-white">Export Professional ATS Resume Template</h3>
                  <p className="text-xs text-gray-400">Convert raw/original PDF content into clean ATS-friendly HTML & PDF format</p>
                </div>
              </div>
              <button
                onClick={() => setShowExportModal(false)}
                className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-white/5 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Action Bar */}
            <div className="px-6 py-3 border-b border-white/5 bg-surface-200/30 flex items-center justify-between flex-wrap gap-3">
              <div className="flex items-center gap-2">
                <button
                  onClick={handleConvertWithAI}
                  disabled={exportingAI}
                  className="px-3.5 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-medium flex items-center gap-2 transition-all shadow-md shadow-indigo-600/20"
                  title="Ask AI model to parse and convert original PDF content into this template"
                >
                  {exportingAI ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Sparkles className="w-3.5 h-3.5 text-yellow-300" />}
                  Ask AI to Convert PDF Content
                </button>

                <button
                  onClick={handleDownloadPDF}
                  className="px-3.5 py-2 rounded-lg bg-white/10 hover:bg-white/15 text-white text-xs font-medium flex items-center gap-2 transition-all"
                >
                  <Download className="w-3.5 h-3.5 text-indigo-400" />
                  Print / Save PDF
                </button>

                <button
                  onClick={handleDownloadHTML}
                  className="px-3.5 py-2 rounded-lg bg-white/10 hover:bg-white/15 text-white text-xs font-medium flex items-center gap-2 transition-all"
                >
                  <FileCode className="w-3.5 h-3.5 text-emerald-400" />
                  Download HTML
                </button>

                <button
                  onClick={handleCopyHTML}
                  className="px-3.5 py-2 rounded-lg bg-white/10 hover:bg-white/15 text-white text-xs font-medium flex items-center gap-2 transition-all"
                >
                  {copySuccess ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5 text-gray-400" />}
                  {copySuccess ? 'Copied HTML!' : 'Copy Code'}
                </button>
              </div>

              {/* Controls & View Switcher */}
              <div className="flex items-center gap-2">
                <button
                  id="btn-modal-fit-single-page"
                  onClick={() => handleToggleFitToSinglePage(!fitToSinglePage)}
                  disabled={isFormattingAI}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-all border ${
                    fitToSinglePage
                      ? 'bg-indigo-600 text-white border-indigo-500 shadow-md shadow-indigo-600/20'
                      : 'bg-surface-300 text-gray-300 border-white/5 hover:bg-surface-400'
                  }`}
                  title="Ask AI to condense and fit resume content strictly onto 1 single page"
                >
                  {isFormattingAI ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
                  ) : (
                    <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
                  )}
                  Fit to 1 Page {fitToSinglePage ? '(Active)' : ''}
                </button>

                <div className="flex items-center bg-surface-300 rounded-lg p-1 border border-white/5">
                  <button
                    onClick={() => setExportTab('preview')}
                    className={`px-3 py-1.5 rounded-md text-xs font-medium flex items-center gap-1.5 transition-colors ${exportTab === 'preview' ? 'bg-indigo-600 text-white shadow' : 'text-gray-400 hover:text-white'}`}
                  >
                    <ExternalLink className="w-3.5 h-3.5" />
                    Visual Preview
                  </button>
                  <button
                    onClick={() => setExportTab('code')}
                    className={`px-3 py-1.5 rounded-md text-xs font-medium flex items-center gap-1.5 transition-colors ${exportTab === 'code' ? 'bg-indigo-600 text-white shadow' : 'text-gray-400 hover:text-white'}`}
                  >
                    <Code className="w-3.5 h-3.5" />
                    HTML Code
                  </button>
                </div>
              </div>
            </div>

            {/* Modal Body Preview */}
            <div className="flex-1 min-h-[450px] overflow-hidden p-4 bg-surface-300/50 flex flex-col">
              {exportTab === 'preview' ? (
                <div className="w-full h-full min-h-[450px] bg-white rounded-xl shadow-inner overflow-hidden border border-gray-200">
                  <iframe
                    title="ATS Template Preview"
                    srcDoc={exportHTML || generatePrintHTML(editorContent)}
                    className="w-full h-full min-h-[450px] border-0"
                  />
                </div>
              ) : (
                <div className="w-full h-full min-h-[450px] bg-gray-950 rounded-xl border border-white/10 p-4 font-mono text-xs text-indigo-300 overflow-auto">
                  <pre className="whitespace-pre-wrap leading-relaxed">{exportHTML || generatePrintHTML(editorContent)}</pre>
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div className="px-6 py-3 border-t border-white/10 bg-surface-200/50 flex items-center justify-between text-xs text-gray-400">
              <span className="flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                Formatted with ATS-friendly Segoe UI font, #2c3e50 headers, flex job headers & 220px skill categories
              </span>
              <button
                onClick={() => setShowExportModal(false)}
                className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/15 text-white text-xs font-medium transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

export default ResumeEditor;
