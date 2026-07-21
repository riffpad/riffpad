"use client";

import { useMemo, useState } from "react";
import { ChevronRight, ChevronDown, FileCode, Folder } from "lucide-react";

interface FileInfo {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
}

interface FileTreeProps {
  files: FileInfo[];
  selectedFile: string | null;
  onSelectFile: (path: string) => void;
}

interface TreeNode {
  name: string;
  path: string;
  isDir: boolean;
  children: TreeNode[];
}

function dirname(path: string): string {
  const idx = path.lastIndexOf("/");
  return idx === -1 ? "" : path.slice(0, idx);
}

function basename(path: string): string {
  const idx = path.lastIndexOf("/");
  return idx === -1 ? path : path.slice(idx + 1);
}

function buildTree(files: FileInfo[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  const root: TreeNode[] = [];

  function getDir(path: string): TreeNode {
    const existing = map.get(path);
    if (existing) return existing;
    const node: TreeNode = {
      name: basename(path),
      path,
      isDir: true,
      children: [],
    };
    map.set(path, node);
    const parentPath = dirname(path);
    if (!parentPath) {
      root.push(node);
    } else {
      getDir(parentPath).children.push(node);
    }
    return node;
  }

  // Create nodes for all explicit entries first.
  for (const file of files) {
    if (file.isDir) {
      getDir(file.path);
    }
  }

  for (const file of files) {
    if (file.isDir) continue;
    const node: TreeNode = {
      name: file.name,
      path: file.path,
      isDir: false,
      children: [],
    };
    const parentPath = dirname(file.path);
    if (!parentPath) {
      root.push(node);
    } else {
      getDir(parentPath).children.push(node);
    }
  }

  return sortNodes(root);
}

function sortNodes(nodes: TreeNode[]): TreeNode[] {
  const sorted = [...nodes].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
  for (const node of sorted) {
    if (node.children.length > 0) {
      node.children = sortNodes(node.children);
    }
  }
  return sorted;
}

export function FileTree({ files, selectedFile, onSelectFile }: FileTreeProps) {
  const tree = useMemo(() => buildTree(files), [files]);
  const [expanded, setExpanded] = useState<Set<string>>(() => {
    // Expand all directories by default so files are visible.
    const set = new Set<string>();
    for (const file of files) {
      if (file.isDir) set.add(file.path);
    }
    return set;
  });

  const toggleDir = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  if (files.length === 0) {
    return <p className="text-xs text-mute px-2 py-1">No files yet</p>;
  }

  return (
    <ul className="space-y-0.5 pb-4">
      {tree.map((node) => (
        <TreeItem
          key={node.path}
          node={node}
          selectedFile={selectedFile}
          onSelectFile={onSelectFile}
          expanded={expanded}
          onToggle={toggleDir}
          depth={0}
        />
      ))}
    </ul>
  );
}

interface TreeItemProps {
  node: TreeNode;
  selectedFile: string | null;
  onSelectFile: (path: string) => void;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  depth: number;
}

function TreeItem({
  node,
  selectedFile,
  onSelectFile,
  expanded,
  onToggle,
  depth,
}: TreeItemProps) {
  const isExpanded = expanded.has(node.path);
  const paddingLeft = 8 + depth * 12;

  if (node.isDir) {
    return (
      <li>
        <button
          onClick={() => onToggle(node.path)}
          className="group flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-card-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          style={{ paddingLeft }}
          aria-expanded={isExpanded}
        >
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 shrink-0 text-mute" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 shrink-0 text-mute" />
          )}
          <Folder className="h-4 w-4 shrink-0 text-mute" />
          <span className="truncate text-body group-hover:text-ink">
            {node.name}
          </span>
        </button>
        {isExpanded && node.children.length > 0 && (
          <ul className="mt-0.5 space-y-0.5">
            {node.children.map((child) => (
              <TreeItem
                key={child.path}
                node={child}
                selectedFile={selectedFile}
                onSelectFile={onSelectFile}
                expanded={expanded}
                onToggle={onToggle}
                depth={depth + 1}
              />
            ))}
          </ul>
        )}
      </li>
    );
  }

  return (
    <li>
      <button
        onClick={() => onSelectFile(node.path)}
        className={`group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 ${
          selectedFile === node.path
            ? "bg-canvas text-ink shadow-sm"
            : "hover:bg-card-soft text-body"
        }`}
        style={{ paddingLeft }}
      >
        <FileCode className="h-4 w-4 shrink-0 text-mute" />
        <span className="truncate">{node.name}</span>
      </button>
    </li>
  );
}
