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

// One cell of an animation row. durationMs overrides the sprite-wide
// Sprite.durationMs for this frame only.
export interface AnimFrame {
  frameId: string;
  durationMs?: number;
}

// One named row of the frame grid. Every frame belongs to exactly one row.
export interface Animation {
  name: string;
  frames: AnimFrame[];
}

export interface Sprite {
  id: string;
  name: string;
  width: number;
  height: number;
  tags: string[];
  // Default hold time per frame during playback, in milliseconds.
  durationMs: number;
  layers: LayerDef[];
  animations: Animation[];
  // Every frame id in grid order (the animation rows flattened).
  frameIds: string[];
  created?: string;
  updated?: string;
}

// The project's single shared palette (GET /api/palette) — every sprite's
// frames index into this same list; there is no more per-sprite palette.
export interface Settings {
  activePreset: string;
  palette: PaletteEntry[];
  updated?: string;
}

export interface PalettePresetInfo {
  id: string;
  name: string;
  colorCount: number;
}

export interface SpriteSummary {
  id: string;
  name: string;
  width: number;
  height: number;
  tags: string[];
  frameCount: number;
  animationNames: string[];
}

export interface SpritePatch {
  name?: string;
  tags?: string[];
  layers?: LayerDef[];
  durationMs?: number;
}

// frames may only reorder a row's existing frames or change their
// durationMs overrides — see sprite.AnimationPatch.
export interface AnimationPatch {
  name?: string;
  frames?: AnimFrame[];
}

export interface ExportFormat {
  name: string;
}
