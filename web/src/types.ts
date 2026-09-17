// Mirrors internal/sprite's JSON boundary exactly (see the plan's
// "Go/JSON boundary" section) — keep these in lockstep with the Go structs.

export interface PixelModifyItem {
  rowIndex: number;
  columnIndex: number;
  color: string; // "" == no pixel, matches dotting's own convention
}

export interface LayerProps {
  id: string;
  data: PixelModifyItem[][];
}

export interface PaletteEntry {
  char: string;
  color: string;
}

export interface LayerDef {
  id: string;
  name: string;
  visible: boolean;
  opacity: number;
}

export type LoopMode = "none" | "forward" | "pingpong";

export interface ClipEntry {
  frameId: string;
  durationMs?: number;
}

export interface Clip {
  name: string;
  loop: LoopMode;
  fps: number;
  entries: ClipEntry[];
}

export interface Sprite {
  id: string;
  name: string;
  width: number;
  height: number;
  tags: string[];
  palette: PaletteEntry[];
  layers: LayerDef[];
  clips: Clip[];
  // Computed at read time from frames/*.px on disk — not stored in
  // sprite.md itself (frame existence has no manifest, see the plan).
  frameIds: string[];
  created?: string;
  updated?: string;
}

export interface SpriteSummary {
  id: string;
  name: string;
  width: number;
  height: number;
  tags: string[];
  frameCount: number;
  clipNames: string[];
}

export interface SpritePatch {
  name?: string;
  tags?: string[];
  layers?: LayerDef[];
}

export interface ClipPatch {
  loop?: LoopMode;
  fps?: number;
  entries?: ClipEntry[];
}

export interface ExportFormat {
  name: string;
}
