import { useMemo } from "react";
import { marked } from "marked";

interface MarkdownContentProps {
  content: string;
}

function normalizeContent(text: string): string {
  // Preserve code blocks, normalize single newlines to double elsewhere
  const parts = text.split(/(```[\s\S]*?```)/);
  return parts
    .map((part, i) => {
      if (i % 2 === 1) return part; // code block
      return part.replace(/(?<!\n)\n(?!\n)/g, "\n\n");
    })
    .join("");
}

export function MarkdownContent({ content }: MarkdownContentProps) {
  const html = useMemo(() => {
    return marked.parse(normalizeContent(content)) as string;
  }, [content]);

  return (
    <div
      className="prose prose-sm max-w-none dark:prose-invert"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
