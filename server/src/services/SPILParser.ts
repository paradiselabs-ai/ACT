/**
 * SPIL Parser — extracts structured data from SPIL-formatted text.
 *
 * MVP scope: extract @-sections and their contents. The format is:
 *   @keyword:         → starts a section (content follows on subsequent lines)
 *   @keyword "value"  → inline key-value
 *   > "text"          → natural language directive
 *   - item            → list item within a section
 *
 * Primary use case: extract @success_criteria for Assurance validation.
 * Full lexer/parser (tokenization, semantic analysis) is deferred.
 */

export interface SPILSection {
  keyword: string;
  value?: string;           // inline value for @keyword "value"
  items: string[];          // list items (- item)
  directives: string[];     // > "text" directives within this section
  raw: string;              // raw text content
}

export interface SPILDocument {
  sections: SPILSection[];
  directives: string[];     // top-level > directives not in a section
}

/**
 * Parse an SPIL-formatted string into structured sections.
 */
export function parseSPIL(text: string): SPILDocument {
  const lines = text.split('\n');
  const document: SPILDocument = { sections: [], directives: [] };
  let currentSection: SPILSection | null = null;

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//')) continue;

    // @keyword "value" — inline key-value
    const inlineMatch = trimmed.match(/^@(\w+)\s+"([^"]*)"$/);
    if (inlineMatch) {
      // Save previous section
      if (currentSection) document.sections.push(currentSection);
      currentSection = {
        keyword: inlineMatch[1],
        value: inlineMatch[2],
        items: [],
        directives: [],
        raw: trimmed,
      };
      continue;
    }

    // @keyword: — section start (with colon)
    const sectionMatch = trimmed.match(/^@(\w+):?\s*$/);
    if (sectionMatch) {
      if (currentSection) document.sections.push(currentSection);
      currentSection = {
        keyword: sectionMatch[1],
        items: [],
        directives: [],
        raw: '',
      };
      continue;
    }

    // @keyword [...] — inline with array/object
    const inlineComplexMatch = trimmed.match(/^@(\w+)\s+(.+)$/);
    if (inlineComplexMatch && !trimmed.startsWith('@success') && !trimmed.startsWith('@error')) {
      if (currentSection) document.sections.push(currentSection);
      currentSection = {
        keyword: inlineComplexMatch[1],
        value: inlineComplexMatch[2],
        items: [],
        directives: [],
        raw: trimmed,
      };
      continue;
    }

    // - item — list item
    const itemMatch = trimmed.match(/^-\s+(.+)$/);
    if (itemMatch) {
      if (currentSection) {
        currentSection.items.push(itemMatch[1]);
        currentSection.raw += '\n' + trimmed;
      }
      continue;
    }

    // > "text" — directive
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

    // Continuation line (belongs to current section)
    if (currentSection) {
      // Could be a sub-key like "key: value" or just content
      currentSection.raw += '\n' + trimmed;
    }
  }

  // Save last section
  if (currentSection) document.sections.push(currentSection);

  return document;
}

/**
 * Extract @success_criteria items from an SPIL document.
 * This is the primary use case for Assurance validation.
 */
export function extractSuccessCriteria(text: string): string[] {
  const doc = parseSPIL(text);
  const section = doc.sections.find(s => s.keyword === 'success_criteria');
  return section?.items ?? [];
}

/**
 * Extract a specific @keyword's value from SPIL text.
 */
export function extractKeyword(text: string, keyword: string): string | undefined {
  const doc = parseSPIL(text);
  const section = doc.sections.find(s => s.keyword === keyword);
  return section?.value;
}

/**
 * Extract all > directives from SPIL text.
 */
export function extractDirectives(text: string): string[] {
  const doc = parseSPIL(text);
  const allDirectives = [...doc.directives];
  for (const section of doc.sections) {
    allDirectives.push(...section.directives);
  }
  return allDirectives;
}

/**
 * Extract @data, @context, @error_handling sub-keys.
 * Returns key-value pairs from indented content.
 */
export function extractSubKeys(text: string, keyword: string): Record<string, string> {
  const doc = parseSPIL(text);
  const section = doc.sections.find(s => s.keyword === keyword);
  if (!section) return {};

  const result: Record<string, string> = {};
  const lines = section.raw.split('\n');

  for (const line of lines) {
    const kvMatch = line.trim().match(/^(\w+):\s+"?([^"]*)"?$/);
    if (kvMatch) {
      result[kvMatch[1]] = kvMatch[2];
    }
  }

  return result;
}
