// SVG icon & escape HTML helpers for resume rendering

export const getContactIconSVG = (item?: string): string => {
  if (!item || typeof item !== 'string') return '';
  const svgStyle = 'width: 12px; height: 12px; max-width: 12px; max-height: 12px; vertical-align: -2px; margin-right: 4px; fill: #000; display: inline-block; flex-shrink: 0;';
  if (item.includes('@')) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M48 64C21.5 64 0 85.5 0 112c0 15.1 7.1 29.3 19.2 38.4L236.8 313.6c11.4 8.5 27 8.5 38.4 0L492.8 150.4c12.1-9.1 19.2-23.3 19.2-38.4c0-26.5-21.5-48-48-48H48zM0 176V384c0 35.3 28.7 64 64 64H448c35.3 0 64-28.7 64-64V176L294.4 339.2c-22.8 17.1-54 17.1-76.8 0L0 176z"/></svg>`;
  }
  if (/[\+\d\(\)\-]{7,}/.test(item)) {
    return `<svg width="12" height="12" viewBox="0 0 512 512" style="${svgStyle}"><path d="M164.9 24.6c-7.7-18.6-28-28.5-47.4-23.2l-88 24C12.1 30.2 0 46 0 64C0 311.4 200.6 512 448 512c18 0 33.8-12.1 38.6-29.5l24-88c5.3-19.4-4.6-39.7-23.2-47.4l-96-40c-16.3-6.8-35.2-2.1-46.3 11.6L304.7 368C234.3 334.7 177.3 277.7 144 207.3L193.3 167c13.7-11.2 18.4-30 11.6-46.3l-40-96z"/></svg>`;
  }
  return `<svg width="12" height="12" viewBox="0 0 384 512" style="${svgStyle}"><path d="M215.7 499.2C267 435 384 279.4 384 192C384 86 298 0 192 0S0 86 0 192c0 87.4 117 243 168.3 307.2c12.3 15.3 35.1 15.3 47.4 0zM192 128a64 64 0 1 1 0 128 64 64 0 1 1 0-128z"/></svg>`;
};

export function escapeHTML(str?: string | null): string {
  if (str === undefined || str === null) return '';
  return String(str).replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}

export function formatBulletActionVerb(str?: string | null): string {
  if (!str || typeof str !== 'string') return '';
  let s = str.replace(/^[•\-▪◦\s]+/, '').replace(/^\*\s+/, '').trim();
  s = escapeHTML(s);

  // Convert markdown **keyword** or <strong>keyword</strong> coming from AI into bold HTML tags
  s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/&lt;strong&gt;(.*?)&lt;\/strong&gt;/gi, '<strong>$1</strong>');
  s = s.replace(/&lt;b&gt;(.*?)&lt;\/b&gt;/gi, '<strong>$1</strong>');

  return s;
}

export function formatJobTitleLine(title?: string | null): string {
  if (!title || typeof title !== 'string') return '';
  if (title.includes('|')) {
    const parts = title.split('|').map(p => p.trim());
    const mainTitle = parts[0];
    const restType = parts.slice(1).join(' | ');
    return `<strong>${escapeHTML(mainTitle)}</strong> | ${escapeHTML(restType)}`;
  }
  return `<strong>${escapeHTML(title)}</strong>`;
}

export function renderFormattedText(str?: string | null): string {
  if (!str || typeof str !== 'string') return '';
  let s = str.replace(/^[•\-▪◦\s]+/, '').replace(/^\*\s+/, '').trim();
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

    // Preserve bold styling in state plain text as **bold**
    if (['STRONG', 'B'].includes(el.tagName)) {
      const trimmed = text.trim();
      if (trimmed) {
        return text.replace(trimmed, `**${trimmed}**`);
      }
    }

    // Add space between flex-between children (e.g. Job Title and Date/Location)
    if (el.classList && el.classList.contains('flex-between')) {
      const childrenText: string[] = [];
      for (let i = 0; i < el.childNodes.length; i++) {
        const childVal = getCleanTextFromDOM(el.childNodes[i]).trim();
        if (childVal) childrenText.push(childVal);
      }
      text = childrenText.join(' ');
    }

    const isBlock = ['P', 'DIV', 'H1', 'H2', 'H3', 'LI', 'TR'].includes(el.tagName);
    if (isBlock && !text.endsWith('\n')) {
      text += '\n';
    }
    return text;
  }
  return '';
};
