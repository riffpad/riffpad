import { memo } from "react";
import { FileCode } from "lucide-react";

interface FileChangeProps {
  path: string;
}

function FileChangeImpl({ path }: FileChangeProps) {
  return (
    <div className="flex items-center gap-2 rounded-md border border-hairline bg-card-doc/80 px-3 py-2 text-xs text-body">
      <FileCode className="h-3.5 w-3.5 text-mute shrink-0" />
      <span className="truncate">{path}</span>
    </div>
  );
}

export const FileChange = memo(FileChangeImpl);
