export interface ParsedJob {
  title: string;
  date: string;
  company: string;
  location: string;
  bullets: string[];
  techStack?: string;
}

export interface ParsedEdu {
  institution: string;
  date: string;
  degree: string;
}

export interface ParsedSkill {
  category: string;
  items: string;
}

export interface ParsedCustomSection {
  title: string;
  content: string[];
}

export interface ParsedResume {
  name: string;
  title: string;
  contactItems: string[];
  summary: string;
  skills: ParsedSkill[];
  workExperience: ParsedJob[];
  education: ParsedEdu[];
  customSections: ParsedCustomSection[];
}

export function parseResumeStructure(text: string): ParsedResume {
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

      // Handle split date across two lines
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

export function convertStructuredToText(sr: any): string {
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
    lines.push('WORK EXPERIENCE');
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
    lines.push('EDUCATION');
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
