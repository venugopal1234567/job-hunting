// SVG icon & escape HTML helpers for resume rendering

export const getContactIconSVG = (item: string): string => {
  const svgStyle = 'width: 12px; height: 12px; max-width: 12px; max-height: 12px; vertical-align: -2px; margin-right: 4px; fill: #000; display: inline-block; flex-shrink: 0;';
  if (item.includes('@')) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M48 64C21.5 64 0 85.5 0 112c0 15.1 7.1 29.3 19.2 38.4L236.8 313.6c11.4 8.5 27 8.5 38.4 0L492.8 150.4c12.1-9.1 19.2-23.3 19.2-38.4c0-26.5-21.5-48-48-48H48zM0 176V384c0 35.3 28.7 64 64 64H448c35.3 0 64-28.7 64-64V176L294.4 339.2c-22.8 17.1-54 17.1-76.8 0L0 176z"/></svg>`;
  }
  if (/[\+\d\(\)\-]{7,}/.test(item)) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M164.9 24.6c-7.7-18.6-28-28.5-47.4-23.2l-88 24C12.1 30.2 0 46 0 64C0 311.4 200.6 512 448 512c18 0 33.8-12.1 38.6-29.5l24-88c5.3-19.4-4.6-39.7-23.2-47.4l-96-40c-16.3-6.8-35.2-2.1-46.3 11.6L304.7 368C234.3 334.7 177.3 277.7 144 207.3L193.3 167c13.7-11.2 18.4-30 11.6-46.3l-40-96z"/></svg>`;
  }
  return `<svg width="12" height="12" viewBox="0 0 384 512" style="${svgStyle}"><path d="M215.7 499.2C267 435 384 279.4 384 192C384 86 298 0 192 0S0 86 0 192c0 87.4 117 243 168.3 307.2c12.3 15.3 35.1 15.3 47.4 0zM192 128a64 64 0 1 1 0 128 64 64 0 1 1 0-128z"/></svg>`;
};

export function escapeHTML(str: string): string {
  return str.replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}

export function formatBulletActionVerb(str: string): string {
  if (!str) return '';
  let s = str.replace(/^[•\-▪◦\s]+/, '').trim();
  s = escapeHTML(s);

  // Convert markdown **keyword** or <strong>keyword</strong> coming from AI into bold HTML tags
  s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>');
  s = s.replace(/&lt;b&gt;(.*?)&lt;\/b&gt;/gi, '<strong>$1</strong>');

  return s;
}

export function formatJobTitleLine(title: string): string {
  if (!title) return '';
  if (title.includes('|')) {
    const parts = title.split('|').map(p => p.trim());
    const mainTitle = parts[0];
    const restType = parts.slice(1).join(' | ');
    return `<strong>${escapeHTML(mainTitle)}</strong> | ${escapeHTML(restType)}`;
  }
  return `<strong>${escapeHTML(title)}</strong>`;
}

export function renderFormattedText(str: string): string {
  if (!str) return '';
  let s = str.replace(/^[•\-*▪◦\s]+/, '').trim();
  s = escapeHTML(s);
  s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>');
  s = s.replace(/&lt;b&gt;(.*?)&lt;\/b&gt;/gi, '<strong>$1</strong>');
  return s;
}

export const IMPORTANT_KEYWORDS = [
  'Go', 'Golang', 'Kubernetes', 'Docker', 'Google Pub/Sub', 'Pub/Sub', 'Redis', 'PostgreSQL',
  'DynamoDB', 'MongoDB', 'NATS', 'Helm', 'AWS', 'Azure', 'GCP', 'gRPC', 'Python', 'TypeScript',
  'SQL', 'Shell Scripting', 'Platform Engineering', 'Test-Driven Development', 'TDD', 'System Design',
  'Agile', 'Scrum', 'Agile/Scrum', 'Remote Collaboration', 'CI/CD Automation', 'CI/CD', 'REST API',
  'WebSocket', 'Mocha', 'SLB OSDU Data Platform', 'OSDU'
];

export function formatTextWithHighlights(text: string, highlight: boolean, customKeywords?: string[]): string {
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

export interface MatchResult {
  start: number;
  end: number;
}

export const expandToWordBoundaries = (text: string, start: number, end: number): MatchResult => {
  let newStart = start;
  let newEnd = end;
  const isWordChar = (char: string) => /[a-zA-Z0-9_]/.test(char);

  if (start > 0 && isWordChar(text[start]) && isWordChar(text[start - 1])) {
    while (newStart > 0 && isWordChar(text[newStart - 1])) {
      newStart--;
    }
  }

  if (end < text.length && isWordChar(text[end - 1]) && isWordChar(text[end])) {
    while (newEnd < text.length && isWordChar(text[end])) {
      newEnd++;
    }
  }

  return { start: newStart, end: newEnd };
};

export const findFlexibleMatch = (text: string, pattern: string): MatchResult | null => {
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

export const getCleanTextFromDOM = (node: Node): string => {
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
