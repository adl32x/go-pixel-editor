import { useEffect, useState } from "react";
import * as api from "../api";
import type { BuildResult, Settings } from "../types";

interface BuildSettingsProps {
  settings: Settings | null;
  onSettingsChanged: (settings: Settings) => void;
}

// Where `pixel build` (and the Build button) writes rendered sprites. There
// is deliberately no default — it's wherever the game reads its assets
// from. Each sprite can add its own subfolder (see SpriteMeta).
export default function BuildSettings({ settings, onSettingsChanged }: BuildSettingsProps) {
  const [draft, setDraft] = useState(settings?.buildOut ?? "");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => setDraft(settings?.buildOut ?? ""), [settings?.buildOut]);

  async function save() {
    if (draft.trim() === (settings?.buildOut ?? "")) return;
    try {
      onSettingsChanged(await api.patchSettings({ buildOut: draft.trim() }));
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="palette-settings build-settings">
      <h4>Build output</h4>
      <p className="palette-settings-hint">
        Folder the Build button writes <code>&lt;sprite&gt;.png</code> + <code>&lt;sprite&gt;.json</code>{" "}
        into, relative to the project root (e.g. <code>assets/images</code>).
      </p>
      <div className="build-settings-row">
        <input
          value={draft}
          placeholder="assets/images"
          onChange={(e) => setDraft(e.target.value)}
          onBlur={save}
          onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
        />
      </div>
      {error && <p className="frame-grid-error">{error}</p>}
    </div>
  );
}

// Formats a build result as one line for the Build button's status text.
export function summarizeBuild(res: BuildResult): string {
  const parts = [`${res.written.length} written`, `${res.unchanged.length} unchanged`];
  if (res.removed.length > 0) parts.push(`${res.removed.length} removed`);
  if (res.skipped.length > 0) parts.push(`${res.skipped.length} skipped (no frames)`);
  return `Built to ${res.out}: ${parts.join(", ")}`;
}
