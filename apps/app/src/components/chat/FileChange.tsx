import { FileCode } from "lucide-react";

interface FileChangeProps {
  path: string;
}

export function FileChange({ path }: FileChangeProps) {
  return (
    <div className="ml-7 flex items-center gap-2 rounded-md border border-hairline bg-card-doc/80 px-3 py-2 text-xs text-body">
      <FileCode className="h-3.5 w-3.5 text-mute shrink-0" />
      <span className="truncate">{path}</span>
    </div>
  );
}
