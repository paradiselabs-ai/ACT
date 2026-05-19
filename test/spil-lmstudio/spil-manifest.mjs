// SPIL manifest + body helpers for lazy-fetch testing.
// Splits a raw .spil document into individual sections by @keyword boundary,
// formats a manifest (section name + 1-line hint) for the lazy arms,
// and provides utilities for the full-inject arms.

/**
 * Parse a SPIL doc text into a Map of section name → body.
 *
 * A section starts at a line matching ^@keyword(\s+|:|$) at column 0 and runs
 * until the next such line or EOF. The body INCLUDES the leading @keyword line
 * so the model sees identical text whether it gets the section via full-inject
 * or via spil_get.
 *
 * @param {string} text — raw SPIL document
 * @returns {{ order: string[], sections: Map<string, string> }}
 */
export function splitSpilSections(text) {
  const lines = text.split('\n');
  const sections = new Map();
  const order = [];
  let currentName = null;
  let currentLines = [];

  const headerRe = /^@(\w+)(\s+.*|:?\s*)$/;

  const flush = () => {
    if (currentName == null) return;
    // trim leading/trailing empty lines from each section for cleaner fetches
    while (currentLines.length && currentLines[0].trim() === '') currentLines.shift();
    while (currentLines.length && currentLines[currentLines.length - 1].trim() === '') currentLines.pop();
    sections.set(currentName, currentLines.join('\n'));
    order.push(currentName);
  };

  for (const line of lines) {
    const m = line.match(headerRe);
    if (m && line.startsWith('@')) {
      flush();
      currentName = m[1];
      currentLines = [line];
    } else if (currentName != null) {
      currentLines.push(line);
    }
    // lines before the first @section (preamble) are intentionally dropped
  }
  flush();

  return { order, sections };
}

/**
 * Format a manifest for the lazy-arm user prompt.
 * Each line: "@<section> — <hint>"
 *
 * @param {string[]} order — ordered section names
 * @param {Object<string,string>} hints — section name → 1-line hint
 * @returns {string}
 */
export function formatManifest(order, hints) {
  const lines = ['MANIFEST (call spil_get with the section name to fetch its body):'];
  for (const name of order) {
    const hint = hints[name] ?? '(no hint)';
    lines.push(`  @${name} — ${hint}`);
  }
  return lines.join('\n');
}

/**
 * Concatenate all sections in order — the full-SPIL arm's user prompt body.
 */
export function assembleFullSpil(order, sections) {
  return order.map(n => sections.get(n)).join('\n\n');
}

/**
 * Count approximate tokens in a string (1 token ≈ 4 chars heuristic).
 * Only for diagnostic logging — actual counts come from LM Studio usage field.
 */
export function approxTokens(text) {
  return Math.ceil((text || '').length / 4);
}
