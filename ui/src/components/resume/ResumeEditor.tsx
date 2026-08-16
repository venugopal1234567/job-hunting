import React, { useEffect, useState, useRef, useCallback } from 'react';
import {
  Download, Save, Upload, Clock, PanelRight, PanelRightClose,
  FileText, AlertCircle, Loader2, CheckCircle2, GitBranch,
  Sparkles, Cpu, Zap, ThumbsUp, ThumbsDown, CornerDownLeft, Check, X, RotateCcw
} from 'lucide-react';
import { Job, ResumeVersion } from '../../types';
import { getResumeFullText, getActiveResume, analyzeJob, uploadResume, exportResumePDF } from '../../services/api';
import { useResumeEditor } from '../../hooks/useResumeEditor';
import ChatPanel from './ChatPanel';
import ATSScoreBar from './ATSScoreBar';
import AppliedDialog from './AppliedDialog';
import ResumeUploader from './ResumeUploader';

const getContactIconSVG = (item: string) => {
  const svgStyle = 'width: 12px; height: 12px; max-width: 12px; max-height: 12px; vertical-align: -2px; margin-right: 4px; fill: #000; display: inline-block; flex-shrink: 0;';
  if (item.includes('@')) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M48 64C21.5 64 0 85.5 0 112c0 15.1 7.1 29.3 19.2 38.4L236.8 313.6c11.4 8.5 27 8.5 38.4 0L492.8 150.4c12.1-9.1 19.2-23.3 19.2-38.4c0-26.5-21.5-48-48-48H48zM0 176V384c0 35.3 28.7 64 64 64H448c35.3 0 64-28.7 64-64V176L294.4 339.2c-22.8 17.1-54 17.1-76.8 0L0 176z"/></svg>`;
  }
  if (/[\+\d\(\)\-]{7,}/.test(item)) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M164.9 24.6c-7.7-18.6-28-28.5-47.4-23.2l-88 24C12.1 30.2 0 46 0 64C0 311.4 200.6 512 448 512c18 0 33.8-12.1 38.6-29.5l24-88c5.3-19.4-4.6-39.7-23.2-47.4l-96-40c-16.3-6.8-35.2-2.1-46.3 11.6L304.7 368C234.3 334.7 177.3 277.7 144 207.3L193.3 167c13.7-11.2 18.4-30 11.6-46.3l-40-96z"/></svg>`;
  }
  return `<svg width="12" height="12" viewBox="0 0 384 512" style="${svgStyle}"><path d="M215.7 499.2C267 435 384 279.4 384 192C384 86 298 0 192 0S0 86 0 192c0 87.4 117 243 168.3 307.2c12.3 15.3 35.1 15.3 47.4 0zM192 128a64 64 0 1 1 0 128 64 64 0 1 1 0-128z"/></svg>`;
};

function escapeHTML(str: string): string {
  return str.replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}

function formatBulletActionVerb(str: string): string {
  if (!str) return '';
  let s = str.replace(/^[•\-*▪◦\s]+/, '').trim();
  s = escapeHTML(s);

  if (/^\*\*(.*?)\*\*/.test(s)) {
    return s.replace(/^\*\*(.*?)\*\*/, '<strong>$1</strong>')
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  }
  if (/^&lt;strong&gt;(.*?)&lt;\/strong&gt;/i.test(s)) {
    return s.replace(/^&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>')
            .replace(/&lt;b&gt;(.*?)&lt;\/b&gt;/gi, '<strong>$1</strong>');
  }

  s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>');

  const match = s.match(/^([A-Za-z0-9\-\/]+)(\s+[\s\S]*)?$/);
  if (match) {
    const firstWord = match[1];
    const rest = match[2] || '';
    if (/^[A-Za-z]/.test(firstWord) && !firstWord.startsWith('<strong>')) {
      return `<strong>${firstWord}</strong>${rest}`;
    }
  }

  return s;
}

function formatJobTitleLine(title: string): string {
  if (!title) return '';
  if (title.includes('|')) {
    const parts = title.split('|').map(p => p.trim());
    const mainTitle = parts[0];
    const restType = parts.slice(1).join(' | ');
    return `<strong>${escapeHTML(mainTitle)}</strong> | ${escapeHTML(restType)}`;
  }
  return `<strong>${escapeHTML(title)}</strong>`;
}

function renderFormattedText(str: string): string {
  if (!str) return '';
  let s = str.replace(/^[•\-*▪◦\s]+/, '').trim();
  s = escapeHTML(s);
  s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>');
  s = s.replace(/&lt;b&gt;(.*?)&lt;\/b&gt;/gi, '<strong>$1</strong>');
  return s;
}

// Helper to parse plain text resume into HTML representation matching exact user template
const generatePrintHTML = (text: string, fitToPage: boolean = false): string => {
  if (!text) return '';

  const parsed = parseResumeStructure(text);

  let bodyHtml = `
    <header>
        <h1>${escapeHTML(parsed.name || 'VENUGOPAL HEGDE')}</h1>
        ${parsed.title ? `<div class="subtitle"><em>${escapeHTML(parsed.title)}</em></div>` : ''}
        <div class="contact-info">
            ${parsed.contactItems.map(item => `
                <span>
                    ${getContactIconSVG(item)}
                    ${escapeHTML(item)}
                </span>
            `).join('')}
        </div>
    </header>
  `;

  if (parsed.summary) {
    bodyHtml += `
    <section>
        <h2>PROFESSIONAL SUMMARY</h2>
        <p>${renderFormattedText(parsed.summary)}</p>
    </section>`;
  }

  if (parsed.workExperience.length > 0) {
    bodyHtml += `
    <section>
        <h2>WORK EXPERIENCES</h2>
        ${parsed.workExperience.map(job => `
            <div class="job-title-container flex-between">
                <div class="job-title">${formatJobTitleLine(job.title)}</div>
                <div class="job-date">${escapeHTML(job.date)}</div>
            </div>
            ${(job.company || job.location) ? `
            <div class="company-container flex-between">
                <div class="company-name"><em>${renderFormattedText(job.company)}</em></div>
                <div class="job-location"><em>${renderFormattedText(job.location)}</em></div>
            </div>` : ''}
            <ul>
                ${job.bullets.map(b => `<li>${formatBulletActionVerb(b)}</li>`).join('')}
            </ul>
            ${job.techStack ? `<div class="tech-used"><em>Technologies / Skills Used : ${renderFormattedText(job.techStack)}</em></div>` : ''}
        `).join('')}
    </section>`;
  }

  if (parsed.education.length > 0) {
    bodyHtml += `
    <section>
        <h2>EDUCATIONS</h2>
        ${parsed.education.map(edu => `
            <div class="flex-between" style="font-size: ${fitToPage ? '13.5px' : '14.5px'}; font-family: 'Times New Roman', Times, serif;">
                <div><strong>${renderFormattedText(edu.institution)}</strong></div>
                <div>${escapeHTML(edu.date)}</div>
            </div>
            <div class="edu-details"><em>${renderFormattedText(edu.degree)}</em></div>
        `).join('')}
    </section>`;
  }

  if (parsed.skills.length > 0) {
    bodyHtml += `
    <section>
        <h2>SKILLS</h2>
        <table class="skills-table">
            ${parsed.skills.map(skill => {
              const cat = skill.category.replace(/:$/, '').trim();
              return `
              <tr>
                  <td><strong>${renderFormattedText(cat)} :</strong></td>
                  <td>${renderFormattedText(skill.items)}</td>
              </tr>`;
            }).join('')}
        </table>
    </section>`;
  }

  parsed.customSections.forEach(sec => {
    bodyHtml += `
    <section>
        <h2>${escapeHTML(sec.title)}</h2>
        ${sec.content.map(c => `<p>${renderFormattedText(c)}</p>`).join('')}
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
            margin: ${fitToPage ? '12px 18px' : '18px 20px'};
        }
        body {
            font-family: "Times New Roman", Times, serif;
            color: #000;
            line-height: ${fitToPage ? '1.2' : '1.25'};
            max-width: 850px;
            margin: 0 auto;
            padding: ${fitToPage ? '10px 15px' : '18px 20px'};
            font-size: ${fitToPage ? '13px' : '13.5px'};
            background-color: #fff;
        }
        
        /* Header Styling */
        header {
            text-align: center;
            margin-bottom: ${fitToPage ? '8px' : '12px'};
        }
        h1 {
            font-family: "Times New Roman", Times, serif;
            font-size: ${fitToPage ? '24px' : '26px'};
            font-weight: bold;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin: 0 0 3px 0;
            text-align: center;
        }
        .subtitle {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: ${fitToPage ? '14px' : '15px'};
            margin: ${fitToPage ? '0 0 4px 0' : '0 0 6px 0'};
            text-align: center;
        }
        .contact-info {
            font-family: "Times New Roman", Times, serif;
            font-size: ${fitToPage ? '12px' : '13px'};
            display: flex;
            justify-content: center;
            align-items: center;
            flex-wrap: wrap;
            gap: 12px;
            text-align: center;
            margin-top: 4px;
        }
        .contact-info span {
            display: inline-flex;
            align-items: center;
        }
        .contact-info svg {
            width: 12px !important;
            height: 12px !important;
            min-width: 12px !important;
            min-height: 12px !important;
            max-width: 12px !important;
            max-height: 12px !important;
            margin-right: 4px;
            vertical-align: -1px;
            fill: #000;
            flex-shrink: 0;
            display: inline-block;
        }

        /* Section Headings */
        h2 {
            font-family: "Times New Roman", Times, serif;
            font-size: ${fitToPage ? '14.5px' : '15.5px'};
            text-transform: uppercase;
            font-weight: bold;
            border-bottom: 1.5px solid #000;
            border-top: none;
            padding-bottom: 2px;
            margin-top: ${fitToPage ? '8px' : '12px'};
            margin-bottom: ${fitToPage ? '5px' : '6px'};
        }

        /* General Content Styling */
        p {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 8px 0;
            font-size: ${fitToPage ? '13px' : '13.5px'};
            text-align: justify;
        }
        
        .flex-between {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
        }

        /* Work Experience Styling */
        .job-title-container {
            margin-bottom: 1px;
            font-size: ${fitToPage ? '13.5px' : '14.5px'};
        }
        .job-title {
            font-family: "Times New Roman", Times, serif;
            font-size: ${fitToPage ? '13.5px' : '14.5px'};
        }
        .job-title strong {
            font-weight: bold;
        }
        .job-date {
            font-family: "Times New Roman", Times, serif;
            font-size: ${fitToPage ? '13px' : '13.5px'};
        }
        .company-container {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: ${fitToPage ? '13px' : '13.5px'};
            margin-bottom: ${fitToPage ? '3px' : '4px'};
        }
        .company-name, .job-location {
            font-style: italic;
        }

        ul {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 4px 0;
            padding-left: 20px;
            font-size: ${fitToPage ? '13px' : '13.5px'};
            text-align: justify;
            list-style-type: disc !important;
        }
        li {
            font-family: "Times New Roman", Times, serif;
            margin-bottom: ${fitToPage ? '2px' : '3px'};
            line-height: ${fitToPage ? '1.2' : '1.28'};
            list-style-type: disc !important;
        }
        li strong {
            font-weight: bold;
        }

        .tech-used {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: ${fitToPage ? '12.5px' : '13px'};
            margin-top: 3px;
            margin-bottom: ${fitToPage ? '6px' : '10px'};
        }
        .tech-used em {
            font-style: italic;
        }

        /* Education */
        .edu-details {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: ${fitToPage ? '13px' : '13.5px'};
            margin-top: 1px;
            margin-bottom: 5px;
        }

        /* Skills Table */
        .skills-table {
            font-family: "Times New Roman", Times, serif;
            width: 100%;
            font-size: ${fitToPage ? '13px' : '13.5px'};
            border-collapse: collapse;
            margin-bottom: 6px;
        }
        .skills-table td {
            vertical-align: top;
            padding: 2.5px 0;
        }
        .skills-table td:first-child {
            font-weight: bold;
            width: 26%;
            padding-right: 8px;
        }
        @media print {
            body { padding: 0; margin: 0; max-width: 100%; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
            .job-title-container, .company-container, section { page-break-inside: avoid; }
        }
    </style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
};

const renderEditorCanvasHTML = (text: string, fitToPage: boolean = false): string => {
  if (!text) return '';
  if (text.includes('<header>') || text.includes('<section>') || text.includes('skills-table') || text.includes('<!DOCTYPE html>')) {
    const match = text.match(/<body>([\s\S]*)<\/body>/i);
    return match ? match[1] : text;
  }
  const full = generatePrintHTML(text, fitToPage);
  const match = full.match(/<body>([\s\S]*)<\/body>/i);
  return match ? match[1] : full;
};

// Helper utilities for parser & HTML formatting
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
  result.name = lines[0].replace(/\[cite:\s*\d+\]/g, '').trim();
  let idx = 1;

  while (idx < lines.length) {
    const line = lines[idx].replace(/\[cite:\s*\d+\]/g, '').trim();
    const isSectionHeader = /^(PROFESSIONAL SUMMARY|SUMMARY|WORK EXPERIENCES|WORK EXPERIENCE|EXPERIENCE|SKILLS|TECHNICAL SKILLS|EDUCATIONS|EDUCATION|PROJECTS|CERTIFICATIONS)$/i.test(line);
    if (isSectionHeader) break;

    const hasEmail = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/.test(line);
    const hasPhone = /(?:\+?\d{1,3}[\s-]?)?\(?\d{3,5}\)?[\s-]?\d{3,5}[\s-]?\d{3,5}/.test(line);

    if (!result.title && !hasEmail && !hasPhone && !/[|•]/.test(line) && line.length < 80) {
      result.title = line;
    } else {
      let workLine = line;

      // Extract title if glued to contact string
      const titleMatch = workLine.match(/^([A-Za-z\s,\(\)\/-]+?(?:Engineer|Developer|Architect|Manager|Lead|Specialist|Consultant|Scientist|Designer|Analyst|Programmer))\s*(?=\+?\d|[\w.-]+@|[A-Z][a-z]+,|$)/i);
      if (titleMatch && !result.title) {
        result.title = titleMatch[1].trim();
        workLine = workLine.substring(titleMatch[0].length).trim();
      }

      // Extract Email
      const emailMatch = workLine.match(/([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/);
      if (emailMatch) {
        result.contactItems.push(emailMatch[1]);
        workLine = workLine.replace(emailMatch[1], ' ').trim();
      }

      // Extract Phone
      const phoneMatch = workLine.match(/(\+?\d{1,3}[\s-]?\d{10,12}|\+?\d{1,4}[\s-]?\d{3,5}[\s-]?\d{3,5})/);
      if (phoneMatch) {
        const cleanPhone = phoneMatch[1].replace(/(\+\d{2})(\d{10})/, '$1 $2');
        result.contactItems.push(cleanPhone);
        workLine = workLine.replace(phoneMatch[1], ' ').trim();
      }

      // Remaining contact parts
      const remainingParts = workLine.split(/[|•]/).map(p => p.trim()).filter(Boolean);
      remainingParts.forEach(p => {
        if (p && !result.contactItems.includes(p)) {
          result.contactItems.push(p);
        }
      });
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
        const catTrim = cat.trim();
        const itemsTrim = items.trim();
        if (itemsTrim) {
          result.skills.push({ category: catTrim, items: itemsTrim });
        } else {
          result.skills.push({ category: catTrim, items: '' });
        }
      } else if (result.skills.length > 0 && !result.skills[result.skills.length - 1].items) {
        result.skills[result.skills.length - 1].items = line.replace(/^[•\-*▪◦\s]+/, '').trim();
      } else {
        const txt = line.replace(/^[•\-*▪◦\s]+/, '').trim();
        result.skills.push({ category: 'Key Skills', items: txt });
      }
    } else if (currentSection === 'WORK') {
      const fullRangeRx = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\u2013\u2014]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))/i;

      if (line.toLowerCase().startsWith('technologies') || line.toLowerCase().startsWith('technologies / skills used')) {
        if (currentJob) {
          const tech = line.replace(/^technologies\s*\/\s*skills\s*used\s*:\s*/i, '').replace(/^technologies\s*used\s*:\s*/i, '').trim();
          currentJob.techStack = tech;
        }
        continue;
      }

      const fullDateMatch = line.match(fullRangeRx);
      const endsWithDateMatch = line.match(/((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4}))\s*$/i);

      // Handle split date across two lines (e.g. Line 1: Portability Engineer ... Jun 2021, Line 2: EPAM Systems May 2023)
      if (currentJob && currentJob.date && !currentJob.date.includes('-') && !currentJob.date.includes('–') && !currentJob.date.includes('Present') && endsWithDateMatch) {
        const endDate = endsWithDateMatch[1];
        currentJob.date = `${currentJob.date} - ${endDate}`;
        const comp = line.replace(/((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4}))\s*$/i, '').trim();
        if (comp) currentJob.company = comp;
        continue;
      }

      // Handle job created with title on prev line, date and company on current line
      if (currentJob && currentJob.title && !currentJob.date && !currentJob.company && fullDateMatch) {
        currentJob.date = fullDateMatch[1];
        currentJob.company = line.replace(fullRangeRx, '').trim().replace(/[\s|–—-]+$/, '').trim();
        continue;
      }

      if (fullDateMatch && line.length < 130 && !/^[•\-*▪◦]/.test(line)) {
        if (currentJob) { result.workExperience.push(currentJob); }
        const date = fullDateMatch[1];
        const title = line.replace(fullRangeRx, '').trim().replace(/[\s|–—-]+$/, '').trim();
        currentJob = { title, date, company: '', location: '', bullets: [] };
      } else if (endsWithDateMatch && line.length < 100 && !currentJob?.bullets.length && !/^[•\-*▪◦]/.test(line)) {
        if (currentJob) { result.workExperience.push(currentJob); }
        const date = endsWithDateMatch[1];
        const title = line.replace(/((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4}))\s*$/i, '').trim().replace(/[\s|–—-]+$/, '').trim();
        currentJob = { title, date, company: '', location: '', bullets: [] };
      } else if (currentJob && !currentJob.company && (line.includes('|') || line.includes(' - ') || line.includes('–') || line.toLowerCase().includes('full time') || line.toLowerCase().includes('remote'))) {
        const parts = line.split(/[|–-]/).map(p => p.trim()).filter(Boolean);
        if (parts.length >= 2) {
          currentJob.company = parts[0];
          currentJob.location = parts.slice(1).join(' - ');
        } else {
          currentJob.location = line;
        }
      } else if (currentJob && !currentJob.location && (line.toLowerCase().includes('full time') || line.toLowerCase().includes('part time') || line.toLowerCase().includes('remote') || line.includes('India') || line.includes('USA'))) {
        currentJob.location = line;
      } else if (currentJob) {
        const txt = line.replace(/^[•\-*▪◦\s]+/, '').trim();
        if (txt) currentJob.bullets.push(txt);
      } else {
        currentJob = { title: line, date: '', company: '', location: '', bullets: [] };
      }
    } else if (currentSection === 'EDUCATION') {
      const fullRangeRx = /((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\u2013\u2014]*(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)?)/i;
      const dm = line.match(fullRangeRx);

      if (currentEdu && !currentEdu.date && dm) {
        currentEdu.date = dm[1];
        const degreeText = line.replace(fullRangeRx, '').trim();
        if (degreeText) currentEdu.degree = degreeText;
      } else if (dm) {
        if (currentEdu) { result.education.push(currentEdu); }
        const date = dm[1];
        const inst = line.replace(fullRangeRx, '').trim().replace(/[\s|–—-]+$/, '').trim();
        currentEdu = { institution: inst, date, degree: '' };
      } else if (currentEdu) {
        currentEdu.degree = currentEdu.degree ? currentEdu.degree + ' ' + line : line;
      } else {
        currentEdu = { institution: line, date: '', degree: '' };
      }
    } else {
      const cust = result.customSections.find(s => s.title === currentSection);
      if (cust) cust.content.push(line);
    }
  }

  if (currentJob) result.workExperience.push(currentJob);
  if (currentEdu) result.education.push(currentEdu);

  // Filter out dangling empty skill categories
  result.skills = result.skills.filter(s => s.items && s.items.trim());

  // Infer skills if empty
  if (result.skills.length === 0) {
    result.skills = [
      { category: 'Programming Languages', items: 'Go (Golang), Python, TypeScript, SQL, Shell Scripting' },
      { category: 'Cloud, Automation & Infrastructure', items: 'AWS, Azure, GCP, Docker, Kubernetes, Helm, Terraform, CI/CD, Ollama' },
      { category: 'Data & Streaming', items: 'Redis, MongoDB, PostgreSQL, Google Pub/Sub, NATS, Kafka, gRPC, REST APIs, WebSocket' },
      { category: 'Practices & Tools', items: 'Test-Driven Development (TDD), Microservices Architecture, Event-Driven Architecture, System Design, Git, Mocha, SLB OSDU Data Platform' }
    ];
  }

  return result;
}

const formatResumeTextToHTML = (text: string, fitToPage: boolean = false): string => {
  if (!text) return '';
  return renderEditorCanvasHTML(text, fitToPage);
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
    chatMessages,
    isChatLoading,
    isSaving,
    isDirty,
    lastSavedSkills,
    initContent,
    updateContent,
    sendMessage,
    answerGapQuestion,
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
  const [saveMessage, setSaveMessage] = useState('');
  const [activeSubTab, setActiveSubTab] = useState<'editor' | 'pdf'>('editor');
  const [hasPDF, setHasPDF] = useState(false);
  const [activeModel, setActiveModel] = useState<string>('');
  const [fitToSinglePage, setFitToSinglePage] = useState(false);
  const [exportingPDF, setExportingPDF] = useState(false);

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

  // Isolate chat messages on job change (Do NOT auto-run ATS)
  useEffect(() => {
    setChatMessages([]);
  }, [selectedJob?.id, setChatMessages]);

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

  // ATS calculation is ONLY run when manually triggered by the user
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

  const handleRevert = useCallback(async () => {
    if (!window.confirm('Are you sure you want to revert all changes? This will restore the original resume text.')) {
      return;
    }
    const result = await revertContent();
    if (result) {
      setChatMessages([]);
      let textToUse = result.html || result.text;
      if (!result.html && result.parsed) {
        textToUse = convertStructuredToText(result.parsed);
      }
      applyFullResume(textToUse);
      await saveContent(textToUse);

      setSaveMessage('Reverted to Original!');
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
    }
  }, [revertContent, setChatMessages, applyFullResume, saveContent]);

  const handleApplyFullResume = useCallback(async (text: string) => {
    applyFullResume(text);
    await saveContent(text);
  }, [applyFullResume, saveContent]);

  const handleExportPDF = async () => {
    setExportingPDF(true);
    try {
      const blob = await exportResumePDF(editorContent, fitToSinglePage);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'Venugopal_Hegde_Resume.pdf';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to export PDF:', err);
      alert('Failed to generate PDF: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      setExportingPDF(false);
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

  return (
    <div className="w-full">
      {/* ── Top Command Bar (ONLY 7 Options Requested) ────────────────────────── */}
      <div className="glass rounded-xl border border-white/10 p-3 mb-4 no-print shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-3">
          {/* Hidden File Input */}
          <input
            type="file"
            ref={uploadInputRef}
            style={{ display: 'none' }}
            accept=".pdf,.txt"
            onChange={handleDirectUpload}
          />

          {/* 7. ATS Score Bar */}
          <div className="flex-1 min-w-[260px]">
            <ATSScoreBar
              score={atsScore}
              previousScore={prevAtsScore}
              loading={atsLoading}
              jobTitle={selectedJob?.title}
              onReanalyze={selectedJob ? runATS : undefined}
            />
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            {/* Status indicator */}
            {saveMessage && (
              <span className="flex items-center gap-1 text-xs text-emerald-400 font-medium animate-fade-in mr-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> {saveMessage}
              </span>
            )}

            {/* 5 & 6. Visual Canvas and Original PDF Tabs */}
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

            {/* 1. Fit to page */}
            <button
              id="btn-fit-single-page"
              onClick={() => setFitToSinglePage(!fitToSinglePage)}
              className={`text-xs px-3 py-1.5 rounded-lg flex items-center gap-1.5 font-medium transition-all border ${
                fitToSinglePage
                  ? 'bg-indigo-600 text-white border-indigo-400 shadow-md shadow-indigo-600/30'
                  : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
              }`}
              title="Fit resume layout onto 1 single page"
            >
              <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
              Fit to page {fitToSinglePage ? '✓' : ''}
            </button>

            {/* 3. Revert */}
            <button
              id="btn-revert-resume"
              onClick={handleRevert}
              disabled={isSaving}
              className="px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/20 text-xs font-medium flex items-center gap-1.5 disabled:opacity-40 transition-all"
              title="Revert all edits back to original resume text"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              Revert
            </button>

            {/* 2. Export to pdf (calls backend headless Chrome API) */}
            <button
              id="btn-export-pdf"
              onClick={handleExportPDF}
              disabled={exportingPDF}
              className="px-3.5 py-1.5 rounded-lg bg-gradient-to-r from-indigo-600 to-blue-600 hover:from-indigo-500 hover:to-blue-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/30 disabled:opacity-50 transition-all"
              title="Export clean PDF file directly via backend API"
            >
              {exportingPDF ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
              ) : (
                <Download className="w-3.5 h-3.5 text-white" />
              )}
              Export to PDF
            </button>

            {/* 4. AI resume coach */}
            <button
              id="btn-toggle-chat"
              onClick={() => setChatVisible(!chatVisible)}
              className={`p-2 px-3 rounded-lg text-xs font-medium border flex items-center gap-1.5 transition-all ${
                chatVisible
                  ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
                  : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
              }`}
              title={chatVisible ? 'Hide AI Resume Coach' : 'Show AI Resume Coach'}
            >
              {chatVisible ? <PanelRightClose className="w-4 h-4" /> : <PanelRight className="w-4 h-4" />}
              <span>AI Resume Coach</span>
            </button>
          </div>
        </div>
      </div>

      {/* 50/50 Split Workspace Grid */}
      <div className={`grid grid-cols-1 ${chatVisible ? 'lg:grid-cols-2' : 'lg:grid-cols-1'} gap-4 h-[calc(100vh-200px)] min-h-[550px] w-full no-print`}>

        {/* ── 4. AI Resume Coach Sidebar Panel ───────────────────────────── */}
        {chatVisible && (
          <div className="min-w-0 h-full glass rounded-xl border border-white/10 overflow-hidden flex flex-col no-print shadow-xl">
            <ChatPanel
              messages={chatMessages}
              loading={isChatLoading}
              onSend={(text) => sendMessage(text, activeModel)}
              onAnswerGap={answerGapQuestion}
              jobTitle={selectedJob?.title}
              activeModel={activeModel}
              onModelChange={setActiveModel}
              onApplyFullResume={handleApplyFullResume}
            />
          </div>
        )}

        {/* ── 5 & 6. Visual Resume Canvas / Original PDF Viewer Area ─────── */}
        <div className="min-w-0 h-full flex flex-col glass rounded-xl border border-white/10 overflow-hidden resume-editor-pane shadow-2xl relative">
          {/* Canvas Sub-Header */}
          <div className="flex items-center justify-between px-4 py-2 border-b border-white/5 bg-surface-100/70 flex-shrink-0 no-print">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
              <span className="text-xs font-semibold text-gray-200">
                {activeSubTab === 'editor' ? 'Visual Resume Canvas' : 'Original Uploaded PDF Document'}
              </span>
              {fitToSinglePage && (
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 font-medium">
                  1-Page Fit Active
                </span>
              )}
            </div>
            <span className="text-xs font-mono text-gray-400">
              {editorContent.length.toLocaleString()} characters
            </span>
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
                    updateContent(newText);
                  }
                }}
                dangerouslySetInnerHTML={{ 
                  __html: formatResumeTextToHTML(editorContent, fitToSinglePage) 
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
    </div>
  );
};

export default ResumeEditor;
