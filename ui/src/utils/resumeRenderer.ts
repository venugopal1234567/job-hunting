import { parseResumeStructure } from './resumeParser';
import {
  escapeHTML,
  getContactIconSVG,
  formatBulletActionVerb,
  formatJobTitleLine,
  renderFormattedText
} from './resumeHelpers';

export const generatePrintHTML = (text: string, fitToPage: boolean = false): string => {
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

  if (parsed.workExperience.length > 0) {
    bodyHtml += `
    <section>
        <h2>WORK EXPERIENCE</h2>
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
        <h2>EDUCATION</h2>
        ${parsed.education.map(edu => `
            <div class="flex-between" style="font-size: ${fitToPage ? '13.5px' : '14.5px'}; font-family: 'Times New Roman', Times, serif;">
                <div><strong>${renderFormattedText(edu.institution)}</strong></div>
                <div>${escapeHTML(edu.date)}</div>
            </div>
            <div class="edu-details"><em>${renderFormattedText(edu.degree)}</em></div>
        `).join('')}
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
            @page {
                size: letter;
                margin: ${fitToPage ? '4mm 6mm' : '10mm 12mm'};
            }
            body { 
                padding: 0; 
                margin: 0 auto; 
                max-width: 100%; 
                font-size: ${fitToPage ? '8.5pt' : '10pt'} !important; 
                line-height: ${fitToPage ? '1.15' : '1.25'} !important; 
                -webkit-print-color-adjust: exact; 
                print-color-adjust: exact; 
            }
            h1 { font-size: ${fitToPage ? '14pt' : '18pt'} !important; }
            h2 { font-size: ${fitToPage ? '9.5pt' : '11pt'} !important; margin-top: ${fitToPage ? '4px' : '8px'} !important; margin-bottom: ${fitToPage ? '2px' : '4px'} !important; }
            p, li, td { font-size: ${fitToPage ? '8.5pt' : '9.5pt'} !important; }
            ul { margin-bottom: ${fitToPage ? '2px' : '4px'} !important; }
            li { margin-bottom: ${fitToPage ? '1px' : '2px'} !important; }
            .tech-used { margin-bottom: ${fitToPage ? '2px' : '6px'} !important; }
            .skills-table { margin-bottom: ${fitToPage ? '2px' : '6px'} !important; }
            .job-title-container, .company-container { break-inside: avoid; page-break-inside: avoid; }
        }
    </style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
};

export const renderEditorCanvasHTML = (text: string, fitToPage: boolean = false): string => {
  if (!text) return '';
  if (text.includes('<header>') || text.includes('<section>') || text.includes('skills-table') || text.includes('<!DOCTYPE html>')) {
    const match = text.match(/<body>([\s\S]*)<\/body>/i);
    return match ? match[1] : text;
  }
  const full = generatePrintHTML(text, fitToPage);
  const match = full.match(/<body>([\s\S]*)<\/body>/i);
  return match ? match[1] : full;
};

export const formatResumeTextToHTML = (text: string, fitToPage: boolean = false): string => {
  if (!text) return '';
  let fullDoc = text;
  if (!text.includes('<!DOCTYPE html>') && !text.includes('<html')) {
    fullDoc = generatePrintHTML(text, fitToPage);
  }

  const printOverrideCSS = `<style id="print-fit-override">
@media print {
    @page {
        size: letter;
        margin: 10mm 12.7mm !important;
    }
    html, body {
        background: #fff !important;
        color: #000 !important;
        font-family: "Times New Roman", Times, serif !important;
        font-size: 10.5px !important;
        line-height: 1.22 !important;
        padding: 0 !important;
        margin: 0 auto !important;
        max-width: 100% !important;
        -webkit-print-color-adjust: exact !important;
        print-color-adjust: exact !important;
    }
    header { margin-bottom: 4px !important; }
    h1 { font-size: 17px !important; margin: 0 0 1px 0 !important; }
    .subtitle { font-size: 11px !important; margin-bottom: 2px !important; }
    .contact-info { font-size: 9.5px !important; gap: 10px !important; margin-top: 1px !important; }
    .contact-info span { font-size: 9.5px !important; }
    .contact-info svg { width: 9px !important; height: 9px !important; }
    h2 { 
        font-size: 12px !important; 
        margin-top: 6px !important; 
        margin-bottom: 3px !important; 
        padding-bottom: 1px !important;
        border-bottom: 1px solid #000 !important;
        page-break-after: avoid !important;
        break-after: avoid !important;
    }
    p { font-size: 10.5px !important; line-height: 1.22 !important; margin: 0 0 3px 0 !important; }
    ul { margin: 2px 0 4px 0 !important; padding-left: 15px !important; font-size: 10.5px !important; }
    ul li { margin-bottom: 1.5px !important; line-height: 1.22 !important; font-size: 10.5px !important; }
    .job-title-container { margin-bottom: 0px !important; font-size: 10.5px !important; }
    .job-title { font-size: 10.5px !important; }
    .job-date { font-size: 10.5px !important; }
    .company-container { font-size: 10.5px !important; margin-bottom: 1px !important; }
    .tech-used { font-size: 9.5px !important; margin-top: 1px !important; margin-bottom: 4px !important; }
    .edu-details { font-size: 10.5px !important; margin-top: 0px !important; margin-bottom: 3px !important; }
    .skills-table { font-size: 10.5px !important; margin-bottom: 4px !important; width: 100% !important; }
    .skills-table td { padding: 1.5px 0 !important; font-size: 10.5px !important; }
    section, .job-title-container, .company-container { page-break-inside: avoid !important; break-inside: avoid !important; }
}
</style>`;

  if (fullDoc.includes('</head>')) {
    return fullDoc.replace('</head>', printOverrideCSS + '</head>');
  }
  return printOverrideCSS + fullDoc;
};
