package prompts

// ConvertResumePromptTemplate is used to convert raw resume text into structured resume JSON
const ConvertResumePromptTemplate = `You are an expert ATS resume parser and formatter.
Extract all information from the candidate's resume and return it strictly as a JSON object matching this exact schema:

{
  "name": "<Candidate Full Name>",
  "title": "<Candidate Current / Target Job Title, e.g. Senior Software Development Engineer>",
  "contact_items": ["<Phone Number>", "<Email Address>", "<Location (City, State/Country)>"],
  "summary": "<Professional summary paragraph>",
  "work_experience": [
    {
      "title": "<Job Role / Title>",
      "date": "<Start Date - End Date>",
      "company": "<Company Name>",
      "location": "<Location>",
      "bullets": ["<Bullet point 1>", "<Bullet point 2>"],
      "tech_stack": "<Comma-separated technologies/skills used>"
    }
  ],
  "education": [
    {
      "institution": "<College / University Name | Location>",
      "date": "<Year Range, e.g., 2015 - 2019>",
      "degree": "<Degree Name>"
    }
  ],
  "skills": [
    {
      "category": "<Category Name, e.g., Databases, Frameworks & Libraries, Programming Languages, Soft Skills, Tools & Platforms>",
      "items": "<Comma-separated skills in this category>"
    }
  ]
}

CRITICAL FORMATTING CONSTRAINTS:
1. Contact items must be clean strings without pipe ('|') separators.
2. YOU MUST EXTRACT ALL WORK EXPERIENCES AND COMPANIES FROM THE RESUME. DO NOT OMIT OR STRIP ANY JOB ROLE.
3. Job Title must be present for every entry.
4. Company Name must be present for every entry.
5. Technologies / Skills Used must be populated on tech_stack for each job.
6. Skill categories must be organized cleanly.

Remove all citation markers like [cite: 1].%s
CANDIDATE RESUME TEXT:
%s

OUTPUT STRICTLY VALID JSON:`

// SinglePageFitConstraint is the single page instruction appended when fitSinglePage is enabled
const SinglePageFitConstraint = `
CRITICAL SINGLE PAGE FIT & ZERO LOSS CONSTRAINTS:
1. YOU MUST PRESERVE EVERY SINGLE WORK EXPERIENCE ENTRY AND COMPANY FROM THE ORIGINAL RESUME. NEVER STRIP OFF, DELETE, OR OMIT ANY COMPANY OR JOB EXPERIENCE.
2. To fit strictly on 1 single letter page, condense each bullet point into punchy, high-impact single-line achievements. Eliminate filler words, but DO NOT drop entire bullet points or whole job experiences.
3. Categorize skills into clean, standardized categories matching: "Databases", "Frameworks & Libraries", "Programming Languages", "Soft Skills", "Tools & Platforms".
4. Ensure "tech_stack" is populated for every work experience entry.`
