import type { ReactNode } from "react";
import { Link } from "react-router";
import { ArrowLeft } from "lucide-react";

interface AppHeaderProps {
  backHref?: string;
  title: string;
  rightContent?: ReactNode;
}

export function AppHeader({ backHref, title, rightContent }: AppHeaderProps) {
  return (
    <header className="sticky top-0 z-10 flex items-center gap-3 border-b bg-background px-4 py-3">
      {backHref && (
        <Link to={backHref} className="inline-flex items-center justify-center size-8 rounded-lg hover:bg-muted">
          <ArrowLeft className="h-4 w-4" />
        </Link>
      )}
      <h1 className="flex-1 text-lg font-semibold truncate">{title}</h1>
      {rightContent}
    </header>
  );
}
