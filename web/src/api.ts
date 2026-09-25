import type {
  AnimationPatch,
  ExportFormat,
  LayerProps,
  PalettePresetInfo,
  Settings,
  Sprite,
  SpritePatch,
  SpriteSummary,
} from "./types";

const BASE = "/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      // response wasn't JSON — keep statusText
    }
    throw new Error(message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

// -- sprites -----------------------------------------------------------

export function listSprites(): Promise<SpriteSummary[]> {
  return request("/sprites");
}

export function createSprite(input: {
  name: string;
  width: number;
  height: number;
  tags?: string;
}): Promise<Sprite> {
  return request("/sprites", { method: "POST", body: JSON.stringify(input) });
}

export function getSprite(id: string): Promise<Sprite> {
  return request(`/sprites/${id}`);
}

export function patchSprite(id: string, patch: SpritePatch): Promise<Sprite> {
  return request(`/sprites/${id}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

export function deleteSprite(id: string): Promise<void> {
  return request(`/sprites/${id}`, { method: "DELETE" });
}

// -- frames --------------------------------------------------------------

// Appends a new blank frame to the end of the given animation row (the last
// row when omitted).
export function addFrame(spriteId: string, animation?: string): Promise<{ frameId: string }> {
  return request(`/sprites/${spriteId}/frames`, {
    method: "POST",
    body: JSON.stringify({ animation: animation ?? "" }),
  });
}

export function getFrame(spriteId: string, frameId: string): Promise<LayerProps[]> {
  return request(`/sprites/${spriteId}/frames/${frameId}`);
}

export function putFrame(
  spriteId: string,
  frameId: string,
  layers: LayerProps[],
): Promise<void> {
  return request(`/sprites/${spriteId}/frames/${frameId}`, {
    method: "PUT",
    body: JSON.stringify(layers),
  });
}

export function deleteFrame(spriteId: string, frameId: string): Promise<Sprite> {
  return request(`/sprites/${spriteId}/frames/${frameId}`, { method: "DELETE" });
}

// -- layers ----------------------------------------------------------------
// Adding/removing a layer retrofits or strips a block on every existing
// frame (see sprite.AddLayer/DeleteLayer) — unlike rename/reorder/opacity/
// visibility, which don't touch frame files and so just go through
// patchSprite({ layers }) like they already did.

export function addLayer(spriteId: string, name: string): Promise<Sprite> {
  return request(`/sprites/${spriteId}/layers`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function deleteLayer(spriteId: string, layerId: string): Promise<Sprite> {
  return request(`/sprites/${spriteId}/layers/${encodeURIComponent(layerId)}`, {
    method: "DELETE",
  });
}

// -- animations ------------------------------------------------------------
// The rows of the frame grid. Each call returns the updated sprite.

export function createAnimation(spriteId: string, name: string): Promise<Sprite> {
  return request(`/sprites/${spriteId}/animations`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function patchAnimation(
  spriteId: string,
  name: string,
  patch: AnimationPatch,
): Promise<Sprite> {
  return request(`/sprites/${spriteId}/animations/${encodeURIComponent(name)}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

// Deletes the row *and every frame in it*.
export function deleteAnimation(spriteId: string, name: string): Promise<Sprite> {
  return request(`/sprites/${spriteId}/animations/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

// -- export ----------------------------------------------------------------

export function getExportFormats(): Promise<ExportFormat[]> {
  return request("/export-formats");
}

export function exportUrl(spriteId: string, format: string, animation?: string): string {
  const params = new URLSearchParams({ format });
  if (animation) params.set("animation", animation);
  return `${BASE}/sprites/${spriteId}/export?${params.toString()}`;
}

// `version` is a pure cache-buster: the backend ignores it, but changing it
// changes the URL, which is what actually forces the browser to re-fetch an
// <img> whose spriteId/frameId haven't changed even though the flattened
// pixels underneath have (a draw, a layer's visibility/opacity, a palette
// remap) — see App's previewVersion / PreviewPanel.
export function exportFramePngUrl(spriteId: string, frameId: string, version?: number): string {
  const params = new URLSearchParams({ frame: frameId });
  if (version !== undefined) params.set("v", String(version));
  return `${BASE}/sprites/${spriteId}/export.png?${params.toString()}`;
}

// -- palette ---------------------------------------------------------------
// The project's single shared palette — every sprite draws from this same
// list, there is no more per-sprite palette (see internal/sprite Settings).

export function getPalette(): Promise<Settings> {
  return request("/palette");
}

export function listPalettePresets(): Promise<PalettePresetInfo[]> {
  return request("/palette-presets");
}

export function getPalettePreset(id: string): Promise<{ colors: string[] }> {
  return request(`/palette-presets/${encodeURIComponent(id)}`);
}

// Switching the palette (preset or custom colors) remaps every sprite's
// existing pixels to the nearest matching color in the new one — there is
// no endpoint that just overwrites the palette in place (see
// sprite.RemapPalette). This is a bulk, lossy, project-wide operation; the
// caller should warn before invoking it.
export function putPalette(input: { presetId: string } | { colors: string[] }): Promise<Settings> {
  return request("/palette", { method: "PUT", body: JSON.stringify(input) });
}
