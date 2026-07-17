"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  ReactNode,
} from "react";

export type Locale = "zh" | "en";

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

const dictionaries: Record<Locale, Record<string, string>> = {
  zh: {
    "app.name": "Riffpad",
    "app.tagline": "AI 原生代码灵感草稿本。描述一个想法，让它跑起来。",
    "workspace.new": "新工作区",
    "workspace.connected": "已连接",
    "workspace.disconnected": "已断开",
    "workspace.empty": "暂无工作区",
    "prompt.placeholder": "描述你的想法……例如：写一个打印当前时间的 Python 脚本",
    "prompt.send": "发送",
    "files.title": "文件",
    "files.empty": "还没有文件",
    "viewer.empty": "从左侧选择文件查看内容",
    "events.title": "Agent 事件",
    "events.empty": "还没有事件，发送一个提示开始。",
    "theme.light": "浅色",
    "theme.dark": "深色",
    "theme.system": "跟随系统",
    "lang.zh": "中文",
    "lang.en": "English",
  },
  en: {
    "app.name": "Riffpad",
    "app.tagline": "AI-native sketchbook for code. Describe an idea, make it run.",
    "workspace.new": "New workspace",
    "workspace.connected": "connected",
    "workspace.disconnected": "disconnected",
    "workspace.empty": "No workspace",
    "prompt.placeholder": "Describe your idea... e.g. Write a Python script that prints the current time",
    "prompt.send": "Send",
    "files.title": "Files",
    "files.empty": "No files yet",
    "viewer.empty": "Select a file from the sidebar to view its contents.",
    "events.title": "Agent events",
    "events.empty": "No events yet. Send a prompt to start.",
    "theme.light": "Light",
    "theme.dark": "Dark",
    "theme.system": "System",
    "lang.zh": "中文",
    "lang.en": "English",
  },
};

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("zh");

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    if (typeof document !== "undefined") {
      document.documentElement.lang = next === "zh" ? "zh-CN" : "en";
    }
  }, []);

  const t = useCallback(
    (key: string) => {
      return dictionaries[locale][key] ?? key;
    },
    [locale]
  );

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return ctx;
}
