import { useEffect, useState } from "react";
import * as api from "../api";
import type { PalettePresetInfo, Settings } from "../types";

interface PaletteSettingsProps {
  settings: Settings | null;
  onSettingsChanged: (settings: Settings) => void;
}

// The project's one shared palette editor. Every sprite draws from this
// same list (see internal/sprite's Settings) — there is no per-sprite
// palette to edit. Changing it (a different preset, or a custom color list)
// remaps every sprite's existing pixels to the nearest matching color via
// PUT /api/palette (sprite.RemapPalette), a bulk, lossy, project-wide
// operation — hence the confirm() before either action below.
export default function PaletteSettings({ settings, onSettingsChanged }: PaletteSettingsProps) {
  const [presets, setPresets] = useState<PalettePresetInfo[]>([]);
  const [draftColors, setDraftColors] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.listPalettePresets().then(setPresets);
  }, []);

  useEffect(() => {
    if (settings) setDraftColors(settings.palette.map((p) => p.color));
  }, [settings]);

  async function applyPreset(id: string, name: string) {
    if (
      !confirm(
        `Switch to the "${name}" palette? Every sprite's existing pixels will be remapped to the nearest matching color.`,
      )
    ) {
      return;
    }
    setBusy(true);
    try {
      onSettingsChanged(await api.putPalette({ presetId: id }));
    } finally {
      setBusy(false);
    }
  }

  async function applyCustom() {
    if (draftColors.length === 0) return;
    if (
      !confirm(
        "Apply this custom palette? Every sprite's existing pixels will be remapped to the nearest matching color.",
      )
    ) {
      return;
    }
    setBusy(true);
    try {
      onSettingsChanged(await api.putPalette({ colors: draftColors }));
    } finally {
      setBusy(false);
    }
  }

  function updateColor(index: number, color: string) {
    setDraftColors((prev) => prev.map((c, i) => (i === index ? color : c)));
  }

  function removeColor(index: number) {
    setDraftColors((prev) => prev.filter((_, i) => i !== index));
  }

  function addColor() {
    setDraftColors((prev) => [...prev, "#000000"]);
  }

  function moveColor(index: number, dir: -1 | 1) {
    setDraftColors((prev) => {
      const target = index + dir;
      if (target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  }

  const isDirty =
    !settings ||
    draftColors.length !== settings.palette.length ||
    draftColors.some((c, i) => c.toLowerCase() !== settings.palette[i]?.color.toLowerCase());

  return (
    <div className="palette-settings">
      <h3>Project Palette</h3>
      <p className="palette-settings-hint">
        Every sprite shares this one palette. Changing it remaps every sprite's
        existing pixels to the nearest matching color in the new palette —
        there's no automatic undo for this, only git.
      </p>

      <h4>Presets</h4>
      <div className="palette-settings-presets">
        {presets.map((p) => (
          <button
            key={p.id}
            type="button"
            className={"preset-button" + (settings?.activePreset === p.id ? " selected" : "")}
            disabled={busy}
            onClick={() => applyPreset(p.id, p.name)}
          >
            {p.name} ({p.colorCount})
          </button>
        ))}
      </div>

      <h4>Colors</h4>
      <div className="palette-settings-swatches">
        {draftColors.map((color, i) => (
          <div key={i} className="palette-settings-swatch">
            <input
              type="color"
              value={/^#[0-9a-fA-F]{6}$/.test(color) ? color : "#000000"}
              onChange={(e) => updateColor(i, e.target.value)}
              disabled={busy}
            />
            <div className="palette-settings-swatch-actions">
              <button type="button" onClick={() => moveColor(i, -1)} disabled={busy || i === 0}>
                &larr;
              </button>
              <button
                type="button"
                onClick={() => moveColor(i, 1)}
                disabled={busy || i === draftColors.length - 1}
              >
                &rarr;
              </button>
              <button type="button" onClick={() => removeColor(i)} disabled={busy}>
                &times;
              </button>
            </div>
          </div>
        ))}
        <button type="button" className="palette-settings-add" onClick={addColor} disabled={busy}>
          + Color
        </button>
      </div>

      <button type="button" disabled={!isDirty || busy || draftColors.length === 0} onClick={applyCustom}>
        Apply custom palette
      </button>
    </div>
  );
}
