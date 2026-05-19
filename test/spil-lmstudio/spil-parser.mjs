// Port of server/src/services/SPILParser.ts to plain JS for use as ground-truth oracle.
// Keep in sync — any change to SPILParser.ts must be mirrored here.

export function parseSPIL(text) {
  const lines = text.split('\n');
  const document = { sections: [], directives: [] };
  let currentSection = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    const inlineMatch = trimmed.match(/^@(\w+)\s+"([^"]*)"$/);
    if (inlineMatch) {
      if (currentSection) document.sections.push(currentSection);
      currentSection = { keyword: inlineMatch[1], value: inlineMatch[2], items: [], directives: [], raw: trimmed };
      continue;
    }

    const sectionMatch = trimmed.match(/^@(\w+):?\s*$/);
    if (sectionMatch) {
      if (currentSection) document.sections.push(currentSection);
      currentSection = { keyword: sectionMatch[1], items: [], directives: [], raw: '' };
      continue;
    }

    const inlineComplexMatch = trimmed.match(/^@(\w+)\s+(.+)$/);
    if (inlineComplexMatch && !trimmed.startsWith('@success') && !trimmed.startsWith('@error')) {
      if (currentSection) document.sections.push(currentSection);
      currentSection = { keyword: inlineComplexMatch[1], value: inlineComplexMatch[2], items: [], directives: [], raw: trimmed };
      continue;
    }

    const itemMatch = trimmed.match(/^-\s+(.+)$/);
    if (itemMatch) {
      if (currentSection) {
        currentSection.items.push(itemMatch[1]);
        currentSection.raw += '\n' + trimmed;
      }
      continue;
    }

    const directiveMatch = trimmed.match(/^>\s+"?([^"]*)"?$/);
    if (directiveMatch) {
      const text = directiveMatch[1];
      if (currentSection) {
        currentSection.directives.push(text);
        currentSection.raw += '\n' + trimmed;
      } else {
        document.directives.push(text);
      }
      continue;
    }

    if (currentSection) {
      currentSection.raw += '\n' + trimmed;
    }
  }

  if (currentSection) document.sections.push(currentSection);
  return document;
}

export function stripThinking(text) {
  return text
    .replace(/<think>[\s\S]*?<\/think>/g, '')
    .replace(/<thinking>[\s\S]*?<\/thinking>/g, '')
    .trim();
}

export function extractJSON(text) {
  const cleaned = stripThinking(text);
  const fenced = cleaned.match(/```(?:json)?\s*\n?([\s\S]*?)\n?```/);
  const candidate = fenced ? fenced[1] : cleaned;
  const objMatch = candidate.match(/\{[\s\S]*\}/);
  if (!objMatch) return null;
  try {
    return JSON.parse(objMatch[0]);
  } catch {
    return null;
  }
}
