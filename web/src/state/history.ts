import { useSyncExternalStore } from "react";

// Undo/redo depth per sprite, as last reported by the server in the
// X-Pixel-History header (see internal/server/history.go). api.ts feeds
// every response through recordHistoryHeader, so the editor's Undo/Redo
// buttons stay current without asking separately.

export interface HistoryCounts {
  undo: number;
  redo: number;
}

const EMPTY: HistoryCounts = { undo: 0, redo: 0 };
let counts: Record<string, HistoryCounts> = {};
const listeners = new Set<() => void>();

// Parses "undo=3;redo=1" for the sprite a request was about.
export function recordHistoryHeader(spriteId: string, header: string | null) {
  if (!header) return;
  const parsed: HistoryCounts = { ...EMPTY };
  for (const part of header.split(";")) {
    const [key, value] = part.split("=");
    if (key === "undo" || key === "redo") parsed[key] = Number(value) || 0;
  }
  const prev = counts[spriteId];
  if (prev && prev.undo === parsed.undo && prev.redo === parsed.redo) return;
  counts = { ...counts, [spriteId]: parsed };
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useHistoryCounts(spriteId: string | null): HistoryCounts {
  return useSyncExternalStore(subscribe, () => (spriteId && counts[spriteId]) || EMPTY);
}
