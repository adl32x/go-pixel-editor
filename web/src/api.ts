import type {
  Clip,
  ClipPatch,
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

export function addFrame(spriteId: string): Promise<{ frameId: string }> {
  return request(`/sprites/${spriteId}/frames`, { method: "POST" });
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

export function deleteFrame(
  spriteId: string,
  frameId: string,
  force = false,
): Promise<void> {
  const qs = force ? "?force=true" : "";
  return request(`/sprites/${spriteId}/frames/${frameId}${qs}`, {
    method: "DELETE",
  });
}

// -- clips ---------------------------------------------------------------

export function listClips(spriteId: string): Promise<Clip[]> {
  return request(`/sprites/${spriteId}/clips`);
}

export function createClip(
  spriteId: string,
  input: { name: string; loop: string; fps: number },
): Promise<Clip> {
  return request(`/sprites/${spriteId}/clips`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function patchClip(
  spriteId: string,
  name: string,
  patch: ClipPatch,
): Promise<Clip> {
  return request(`/sprites/${spriteId}/clips/${encodeURIComponent(name)}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

export function deleteClip(spriteId: string, name: string): Promise<void> {
  return request(`/sprites/${spriteId}/clips/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

// -- export ----------------------------------------------------------------

export function getExportFormats(): Promise<ExportFormat[]> {
  return request("/export-formats");
}

export function exportUrl(spriteId: string, format: string, clip?: string): string {
  const params = new URLSearchParams({ format });
  if (clip) params.set("clip", clip);
  return `${BASE}/sprites/${spriteId}/export?${params.toString()}`;
}

export function exportFramePngUrl(spriteId: string, frameId: string): string {
  const params = new URLSearchParams({ frame: frameId });
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
