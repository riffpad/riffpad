"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useLanguage } from "../LanguageProvider";

const mcpServersData = [
  {
    id: "github",
    name: "GitHub",
    desc: "Sync repos, issues and pull requests",
    status: "connected",
  },
  {
    id: "linear",
    name: "Linear",
    desc: "Connect project issues and cycles",
    status: "connected",
  },
  {
    id: "notion",
    name: "Notion",
    desc: "Read and write Notion pages",
    status: "disconnected",
  },
];

export function SkillsMockup() {
  const { t } = useLanguage();
  const s = t.skillsMockup;
  const [activeSetting, setActiveSetting] = useState<"skills" | "mcp">("skills");
  const [skills, setSkills] = useState([
    {
      id: "frontend-design",
      name: "frontend-design",
      desc: "Generate polished frontend interfaces",
      active: true,
    },
    {
      id: "ppt-generation",
      name: "ppt-generation",
      desc: "Create presentation decks from text",
      active: true,
    },
    {
      id: "docx",
      name: "docx",
      desc: "Generate and edit Word documents",
      active: false,
    },
  ]);

  const toggleSkill = (id: string) => {
    setSkills((prev) =>
      prev.map((skill) =>
        skill.id === id ? { ...skill, active: !skill.active } : skill
      )
    );
  };

  const [mcpServers, setMcpServers] = useState(mcpServersData);

  const toggleMcp = (id: string) => {
    setMcpServers((prev) =>
      prev.map((server) =>
        server.id === id
          ? {
              ...server,
              status:
                server.status === "connected" ? "disconnected" : "connected",
            }
          : server
      )
    );
  };

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="grid w-full grid-cols-1 gap-4 lg:grid-cols-[1fr_2fr]"
    >
      {/* Settings pane */}
      <div className="flex min-h-[320px] overflow-hidden rounded-xl border border-hairline bg-surface shadow-xl">
        {/* Sidebar */}
        <div className="flex w-12 flex-col border-r border-hairline bg-surface-soft py-2">
          <button
            onClick={() => setActiveSetting("skills")}
            className={`px-2 py-4 text-[10px] font-semibold leading-tight transition ${
              activeSetting === "skills"
                ? "text-foreground"
                : "text-muted hover:text-foreground"
            }`}
          >
            Skills
          </button>
          <button
            onClick={() => setActiveSetting("mcp")}
            className={`px-2 py-4 text-[10px] font-semibold leading-tight transition ${
              activeSetting === "mcp"
                ? "text-foreground"
                : "text-muted hover:text-foreground"
            }`}
          >
            MCP
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {activeSetting === "skills" ? (
            <div className="space-y-3">
              <div className="mb-3 text-[10px] font-bold uppercase tracking-wider text-muted">
                Skills
              </div>
              {skills.map((skill) => (
                <div
                  key={skill.id}
                  className="flex min-h-[78px] flex-col justify-between rounded-lg border border-hairline bg-surface-doc p-3"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-foreground">
                      {skill.name}
                    </span>
                    <button
                      type="button"
                      onClick={() => toggleSkill(skill.id)}
                      className={`h-5 w-9 rounded-full border p-0.5 transition ${
                        skill.active
                          ? "border-transparent bg-accent"
                          : "border-hairline bg-surface-soft"
                      }`}
                    >
                      <motion.div
                        initial={false}
                        animate={{ x: skill.active ? 16 : 0 }}
                        transition={{ type: "spring", stiffness: 500, damping: 30 }}
                        className={`h-4 w-4 rounded-full shadow-sm ${
                          skill.active ? "bg-surface" : "bg-muted"
                        }`}
                      />
                    </button>
                  </div>
                  <p className="mt-1 text-[10px] leading-relaxed text-muted">
                    {skill.desc}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              <div className="mb-3 text-[10px] font-bold uppercase tracking-wider text-muted">
                MCP
              </div>
              {mcpServers.map((server) => (
                <div
                  key={server.id}
                  className="flex min-h-[78px] flex-col justify-between rounded-lg border border-hairline bg-surface-doc p-3"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-foreground">
                      {server.name}
                    </span>
                    <button
                      type="button"
                      onClick={() => toggleMcp(server.id)}
                      className={`rounded-full px-2.5 py-1 text-[10px] font-semibold transition ${
                        server.status === "connected"
                          ? "bg-accent-green/10 text-accent-green"
                          : "bg-surface-soft text-muted hover:text-foreground"
                      }`}
                    >
                      {server.status === "connected"
                        ? "Connected"
                        : "Connect"}
                    </button>
                  </div>
                  <p className="mt-1 text-[10px] leading-relaxed text-muted">
                    {server.desc}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Chat pane */}
      <div className="flex flex-col overflow-hidden">
        <div className="flex-1 space-y-4 overflow-auto p-4 text-sm">
          <div className="flex justify-end">
            <div className="max-w-[85%] rounded-2xl rounded-br-sm border border-hairline bg-surface-doc px-4 py-2.5 text-xs text-body">
              <div>{s.userMessageShort}...</div>
              <div className="mt-0.5 cursor-pointer text-[10px] text-muted transition hover:text-foreground">
                {s.showMore}
              </div>
            </div>
          </div>

          <div className="flex flex-col items-start gap-2">
            <StatusLine label={s.thinking} />
            <StatusLine label={`${s.runCommandLabel}: ${s.runCommand}`} />
          </div>

          <div className="text-xs text-body">
            {s.assistantReply}{" "}
            <span className="font-mono font-semibold text-foreground">
              {s.assistantReplyName}
            </span>{" "}
            {s.assistantReplySuffix}
          </div>
        </div>
      </div>
    </motion.div>
  );
}

function StatusLine({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted">
      <span className="h-1.5 w-1.5 rounded-full bg-accent" />
      {label}
    </div>
  );
}
