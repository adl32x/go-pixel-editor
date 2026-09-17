import { useEffect, useState } from "react";
import * as api from "../api";
import type { Clip, ExportFormat, LoopMode } from "../types";

interface ClipEditorProps {
  spriteId: string;
  clips: Clip[];
  selectedClipName: string | null;
  selectedFrameId: string | null;
  onSelectClip: (name: string | null) => void;
  onClipsChanged: () => void;
}

export default function ClipEditor({
  spriteId,
  clips,
  selectedClipName,
  selectedFrameId,
  onSelectClip,
  onClipsChanged,
}: ClipEditorProps) {
  const [newName, setNewName] = useState("");
  const [formats, setFormats] = useState<ExportFormat[]>([]);
  const [exportFormat, setExportFormat] = useState("sheet-json");

  useEffect(() => {
    api
      .getExportFormats()
      .then((f) => {
        setFormats(f);
        if (f.length > 0) setExportFormat(f[0].name);
      })
      .catch(() => setFormats([]));
  }, []);

  const clip = clips.find((c) => c.name === selectedClipName) ?? null;

  async function handleCreateClip() {
    if (!newName.trim()) return;
    await api.createClip(spriteId, { name: newName.trim(), loop: "forward", fps: 4 });
    setNewName("");
    onClipsChanged();
    onSelectClip(newName.trim());
  }

  async function handleAddSelectedFrame() {
    if (!clip || !selectedFrameId) return;
    const entries = [...clip.entries, { frameId: selectedFrameId }];
    await api.patchClip(spriteId, clip.name, { entries });
    onClipsChanged();
  }

  async function handleRemoveEntry(index: number) {
    if (!clip) return;
    const entries = clip.entries.filter((_, i) => i !== index);
    await api.patchClip(spriteId, clip.name, { entries });
    onClipsChanged();
  }

  async function handleMoveEntry(index: number, dir: -1 | 1) {
    if (!clip) return;
    const target = index + dir;
    if (target < 0 || target >= clip.entries.length) return;
    const entries = [...clip.entries];
    [entries[index], entries[target]] = [entries[target], entries[index]];
    await api.patchClip(spriteId, clip.name, { entries });
    onClipsChanged();
  }

  async function handleLoopChange(loop: LoopMode) {
    if (!clip) return;
    await api.patchClip(spriteId, clip.name, { loop });
    onClipsChanged();
  }

  async function handleFpsChange(fps: number) {
    if (!clip) return;
    await api.patchClip(spriteId, clip.name, { fps });
    onClipsChanged();
  }

  async function handleDeleteClip() {
    if (!clip) return;
    await api.deleteClip(spriteId, clip.name);
    onSelectClip(null);
    onClipsChanged();
  }

  return (
    <div className="clip-editor">
      <h3>Clips</h3>
      <ul className="clip-list">
        {clips.map((c) => (
          <li key={c.name}>
            <button
              type="button"
              className={"clip-item" + (c.name === selectedClipName ? " selected" : "")}
              onClick={() => onSelectClip(c.name)}
            >
              {c.name} ({c.entries.length})
            </button>
          </li>
        ))}
      </ul>
      <div className="clip-create">
        <input
          placeholder="new clip name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
        />
        <button type="button" onClick={handleCreateClip}>
          + Clip
        </button>
      </div>

      {clip && (
        <div className="clip-detail">
          <div className="clip-detail-header">
            <label>
              loop
              <select
                value={clip.loop}
                onChange={(e) => handleLoopChange(e.target.value as LoopMode)}
              >
                <option value="none">none</option>
                <option value="forward">forward</option>
                <option value="pingpong">pingpong</option>
              </select>
            </label>
            <label>
              fps
              <input
                type="number"
                min={1}
                value={clip.fps}
                onChange={(e) => handleFpsChange(Number(e.target.value))}
              />
            </label>
            <button type="button" onClick={handleDeleteClip}>
              Delete clip
            </button>
          </div>

          <ol className="clip-entries">
            {clip.entries.map((entry, i) => (
              <li key={`${entry.frameId}-${i}`}>
                {entry.frameId}
                {entry.durationMs ? ` @${entry.durationMs}ms` : ""}
                <button type="button" onClick={() => handleMoveEntry(i, -1)}>
                  &uarr;
                </button>
                <button type="button" onClick={() => handleMoveEntry(i, 1)}>
                  &darr;
                </button>
                <button type="button" onClick={() => handleRemoveEntry(i)}>
                  remove
                </button>
              </li>
            ))}
          </ol>
          <button type="button" onClick={handleAddSelectedFrame} disabled={!selectedFrameId}>
            + Add selected frame ({selectedFrameId ?? "none"})
          </button>

          <div className="clip-export">
            <label>
              export format
              <select value={exportFormat} onChange={(e) => setExportFormat(e.target.value)}>
                {formats.map((f) => (
                  <option key={f.name} value={f.name}>
                    {f.name}
                  </option>
                ))}
              </select>
            </label>
            <a
              href={api.exportUrl(spriteId, exportFormat, clip.name)}
              target="_blank"
              rel="noreferrer"
            >
              Export
            </a>
          </div>
        </div>
      )}
    </div>
  );
}
