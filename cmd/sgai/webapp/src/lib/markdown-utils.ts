const FRONTMATTER_RE = /^---\r?\n[\s\S]*?\r?\n---\r?\n?/;

export function stripFrontmatter(content: string): string {
  const trimmed = content.trimStart();
  const match = trimmed.match(FRONTMATTER_RE);
  if (match) {
    return trimmed.slice(match[0].length).trim();
  }
  return trimmed;
}

export function stripMarkdownToPlaintext(markdown: string): string {
  const text = stripFrontmatter(markdown);
  return text
    .replace(/^#{1,6}\s+/gm, "")
    .replace(/\*\*(.+?)\*\*/g, "$1")
    .replace(/__(.+?)__/g, "$1")
    .replace(/\*(.+?)\*/g, "$1")
    .replace(/_(.+?)_/g, "$1")
    .replace(/~~(.+?)~~/g, "$1")
    .replace(/`(.+?)`/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/^>\s?/gm, "")
    .replace(/^- \[[ x]\]\s+/gm, "")
    .replace(/^[-*+]\s+/gm, "")
    .replace(/^\d+\.\s+/gm, "")
    .replace(/^---+$/gm, "")
    .replace(/\n{2,}/g, " ")
    .replace(/\n/g, " ")
    .trim();
}

export function truncateDescription(text: string, maxLen: number = 255): string {
  if (text.length <= maxLen) return text;
  return text.slice(0, maxLen) + "...";
}

export function workspaceDescription(rawGoalContent: string | undefined): string | null {
  if (!rawGoalContent || rawGoalContent.trim().length === 0) return null;
  const plain = stripMarkdownToPlaintext(rawGoalContent);
  if (plain.length === 0) return null;
  return truncateDescription(plain);
}
