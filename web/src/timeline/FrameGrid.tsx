import { useEffect, useState } from "react";
import * as api from "../api";
import type { AnimFrame, Animation, ExportFormat, LayerProps, Sprite } from "../types";
import FrameThumbnail from "./FrameThumbnail";

interface FrameGridProps {
  sprite: Sprite;
  selectedFrameId: string | null;
  selectedAnimation: string | null;
  // Bumped after every pixel save — the selected frame's thumbnail is
  // re-fetched when it changes (other frames can't have changed).
  version: number;
  onSelectFrame: (frameId: string) => void;
  onSelectAnimation: (name: string) => void;
  onSpriteChanged: (sprite: Sprite) => void;
}

// The sprite's frames as a 2D grid: one row per animation, frames left to
// right in playback order. Every frame lives in exactly one row.
export default function FrameGrid({
  sprite,
  selectedFrameId,
  selectedAnimation,
  version,
  onSelectFrame,
  onSelectAnimation,
  onSpriteChanged,
}: FrameGridProps) {
  const [thumbs, setThumbs] = useState<Record<string, LayerProps[]>>({});
  const [formats, setFormats] = useState<ExportFormat[]>([]);
  const [exportFormat, setExportFormat] = useState("sheet-json");
  const [error, setError] = useState<string | null>(null);
  // Drag-and-drop: the frame being dragged, and where it would land — in
  // row `animation`, before frame `before` ("" = end of the row).
  const [dragging, setDragging] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<{ animation: string; before: string } | null>(null);

  useEffect(() => {
    api
      .getExportFormats()
      .then((f) => {
        setFormats(f);
        if (f.length > 0 && !f.some((x) => x.name === "sheet-json")) setExportFormat(f[0].name);
      })
      .catch(() => setFormats([]));
  }, []);

  const frameIdsKey = sprite.frameIds.join(",");
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const entries = await Promise.all(
        sprite.frameIds.map(async (id) => {
          try {
            return [id, await api.getFrame(sprite.id, id)] as const;
          } catch {
            return [id, []] as const;
          }
        }),
      );
      if (!cancelled) setThumbs(Object.fromEntries(entries));
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sprite.id, frameIdsKey]);

  useEffect(() => {
    if (version === 0 || !selectedFrameId) return;
    let cancelled = false;
    api.getFrame(sprite.id, selectedFrameId).then((layers) => {
      if (!cancelled) setThumbs((t) => ({ ...t, [selectedFrameId]: layers }));
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [version]);

  // Runs a mutation, surfacing a server-side validation error (duplicate
  // row name, bad duration) inline instead of as an unhandled rejection.
  async function run(fn: () => Promise<Sprite>): Promise<Sprite | null> {
    try {
      const updated = await fn();
      setError(null);
      onSpriteChanged(updated);
      return updated;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    }
  }

  async function handleAddFrame(animation: string) {
    let frameId: string;
    try {
      ({ frameId } = await api.addFrame(sprite.id, animation));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return;
    }
    await run(() => api.getSprite(sprite.id));
    onSelectFrame(frameId);
  }

  async function handleDuplicateFrame(frameId: string) {
    let dup: string;
    try {
      ({ frameId: dup } = await api.duplicateFrame(sprite.id, frameId));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return;
    }
    await run(() => api.getSprite(sprite.id));
    onSelectFrame(dup);
  }

  function updateDropTarget(animation: string, before: string) {
    if (dropTarget?.animation !== animation || dropTarget.before !== before) {
      setDropTarget({ animation, before });
    }
  }

  // Hovering the left half of a cell targets "before this frame", the right
  // half "before the next one" (or the end of the row).
  function handleCellDragOver(e: React.DragEvent, anim: Animation, index: number) {
    if (!dragging) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = "move";
    const rect = e.currentTarget.getBoundingClientRect();
    const after = e.clientX > rect.left + rect.width / 2;
    const before = after ? (anim.frames[index + 1]?.frameId ?? "") : anim.frames[index].frameId;
    updateDropTarget(anim.name, before);
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    const frameId = dragging;
    const target = dropTarget;
    setDragging(null);
    setDropTarget(null);
    if (!frameId || !target) return;
    // Skip drops that wouldn't change anything: onto itself, or just
    // after itself in its own row.
    const row = sprite.animations.find((a) => a.name === target.animation);
    const i = row?.frames.findIndex((f) => f.frameId === frameId) ?? -1;
    if (i >= 0 && (target.before === frameId || target.before === (row!.frames[i + 1]?.frameId ?? ""))) {
      return;
    }
    run(() => api.moveFrame(sprite.id, frameId, target.animation, target.before));
  }

  async function handleAddAnimation() {
    let n = sprite.animations.length + 1;
    while (sprite.animations.some((a) => a.name === `anim ${n}`)) n++;
    const name = `anim ${n}`;
    if (!(await run(() => api.createAnimation(sprite.id, name)))) return;
    onSelectAnimation(name);
    await handleAddFrame(name);
  }

  function patchFrames(anim: Animation, frames: AnimFrame[]) {
    return run(() => api.patchAnimation(sprite.id, anim.name, { frames }));
  }

  function handleMoveFrame(anim: Animation, index: number, dir: -1 | 1) {
    const target = index + dir;
    if (target < 0 || target >= anim.frames.length) return;
    const frames = [...anim.frames];
    [frames[index], frames[target]] = [frames[target], frames[index]];
    patchFrames(anim, frames);
  }

  function handleFrameDuration(anim: Animation, index: number, durationMs: number | undefined) {
    const frames = anim.frames.map((f, i) => (i === index ? { frameId: f.frameId, durationMs } : f));
    patchFrames(anim, frames);
  }

  function handleDeleteAnimation(anim: Animation) {
    const n = anim.frames.length;
    if (n > 0 && !confirm(`Delete "${anim.name}" and its ${n} frame(s)?`)) return;
    run(() => api.deleteAnimation(sprite.id, anim.name));
  }

  return (
    <div className="frame-grid">
      <div className="frame-grid-toolbar">
        <label>
          Frame duration
          <DurationInput
            value={sprite.durationMs}
            required
            onCommit={(ms) => {
              if (ms && ms !== sprite.durationMs) run(() => api.patchSprite(sprite.id, { durationMs: ms }));
            }}
          />
          ms
        </label>
        <span className="frame-grid-spacer" />
        <label>
          Export
          <select value={exportFormat} onChange={(e) => setExportFormat(e.target.value)}>
            {formats.map((f) => (
              <option key={f.name} value={f.name}>
                {f.name}
              </option>
            ))}
          </select>
        </label>
        <a href={api.exportUrl(sprite.id, exportFormat)} target="_blank" rel="noreferrer">
          All animations
        </a>
      </div>

      {error && <p className="frame-grid-error">{error}</p>}

      <div className="frame-grid-rows">
        {sprite.animations.map((anim, rowIndex) => (
          <div
            key={rowIndex}
            className={
              "frame-grid-row" +
              (anim.name === selectedAnimation ? " selected" : "") +
              (dragging && dropTarget?.animation === anim.name ? " drop-row" : "")
            }
            onClick={() => onSelectAnimation(anim.name)}
            onDragOver={(e) => {
              if (!dragging) return;
              e.preventDefault();
              updateDropTarget(anim.name, "");
            }}
            onDragLeave={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDropTarget(null);
            }}
            onDrop={handleDrop}
          >
            <div className="frame-grid-row-header">
              <AnimationNameInput
                // Remount once a rename lands so the field re-seeds from the
                // server's name (or reverts after a rejected rename).
                key={anim.name}
                name={anim.name}
                onRename={(name) => run(() => api.patchAnimation(sprite.id, anim.name, { name }))}
              />
              <div className="frame-grid-row-actions">
                <a
                  href={api.exportUrl(sprite.id, exportFormat, anim.name)}
                  target="_blank"
                  rel="noreferrer"
                  title={`Export "${anim.name}" as ${exportFormat}`}
                >
                  Export
                </a>
                <button
                  type="button"
                  title="Delete animation and its frames"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDeleteAnimation(anim);
                  }}
                >
                  Delete
                </button>
              </div>
            </div>

            <div className="frame-grid-cells">
              {anim.frames.map((f, i) => {
                const selected = f.frameId === selectedFrameId;
                const dropBefore =
                  dragging !== null &&
                  dropTarget?.animation === anim.name &&
                  dropTarget.before === f.frameId;
                return (
                  <div
                    key={f.frameId}
                    className={
                      "frame-grid-cell" +
                      (selected ? " selected" : "") +
                      (f.frameId === dragging ? " dragging" : "") +
                      (dropBefore ? " drop-before" : "")
                    }
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.setData("text/plain", f.frameId);
                      e.dataTransfer.effectAllowed = "move";
                      setDragging(f.frameId);
                    }}
                    onDragEnd={() => {
                      setDragging(null);
                      setDropTarget(null);
                    }}
                    onDragOver={(e) => handleCellDragOver(e, anim, i)}
                  >
                    <button
                      type="button"
                      className="frame-grid-frame"
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectFrame(f.frameId);
                      }}
                    >
                      <FrameThumbnail
                        layers={thumbs[f.frameId] ?? []}
                        width={sprite.width}
                        height={sprite.height}
                      />
                      <span className="frame-grid-frame-label">{f.frameId}</span>
                    </button>
                    <span
                      className="frame-grid-frame-duplicate"
                      role="button"
                      tabIndex={0}
                      title="Duplicate frame"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDuplicateFrame(f.frameId);
                      }}
                    >
                      +
                    </span>
                    <span
                      className="frame-grid-frame-delete"
                      role="button"
                      tabIndex={0}
                      title="Delete frame"
                      onClick={(e) => {
                        e.stopPropagation();
                        run(() => api.deleteFrame(sprite.id, f.frameId));
                      }}
                    >
                      &times;
                    </span>
                    <div className="frame-grid-cell-controls" onClick={(e) => e.stopPropagation()}>
                      <button
                        type="button"
                        title="Move left"
                        disabled={i === 0}
                        onClick={() => handleMoveFrame(anim, i, -1)}
                      >
                        &lsaquo;
                      </button>
                      <DurationInput
                        value={f.durationMs}
                        placeholder={String(sprite.durationMs)}
                        title="Duration override (ms) — empty uses the sprite default"
                        onCommit={(ms) => {
                          if (ms !== f.durationMs) handleFrameDuration(anim, i, ms);
                        }}
                      />
                      <button
                        type="button"
                        title="Move right"
                        disabled={i === anim.frames.length - 1}
                        onClick={() => handleMoveFrame(anim, i, 1)}
                      >
                        &rsaquo;
                      </button>
                    </div>
                  </div>
                );
              })}
              <button
                type="button"
                className={
                  "frame-grid-add-frame" +
                  (dragging && dropTarget?.animation === anim.name && dropTarget.before === ""
                    ? " drop-before"
                    : "")
                }
                title={`Add a frame to "${anim.name}"`}
                onClick={(e) => {
                  e.stopPropagation();
                  onSelectAnimation(anim.name);
                  handleAddFrame(anim.name);
                }}
              >
                +
              </button>
            </div>
          </div>
        ))}
      </div>

      <button type="button" className="frame-grid-add-row" onClick={handleAddAnimation}>
        + Animation
      </button>
    </div>
  );
}

function AnimationNameInput({
  name,
  onRename,
}: {
  name: string;
  onRename: (name: string) => Promise<Sprite | null>;
}) {
  const [value, setValue] = useState(name);
  async function commit() {
    const trimmed = value.trim();
    if (!trimmed) {
      setValue(name);
      return;
    }
    if (trimmed !== name && !(await onRename(trimmed))) setValue(name);
  }
  return (
    <input
      className="frame-grid-row-name"
      value={value}
      aria-label="Animation name"
      onChange={(e) => setValue(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") e.currentTarget.blur();
        if (e.key === "Escape") {
          setValue(name);
          e.currentTarget.blur();
        }
      }}
    />
  );
}

// A milliseconds field that only commits on blur/Enter, so typing "250"
// doesn't fire three saves. Empty commits `undefined` (clear the override)
// unless `required`, in which case it reverts.
function DurationInput({
  value,
  placeholder,
  title,
  required,
  onCommit,
}: {
  value: number | undefined;
  placeholder?: string;
  title?: string;
  required?: boolean;
  onCommit: (ms: number | undefined) => void;
}) {
  const [text, setText] = useState(value === undefined ? "" : String(value));
  useEffect(() => {
    setText(value === undefined ? "" : String(value));
  }, [value]);

  function commit() {
    const revert = () => setText(value === undefined ? "" : String(value));
    const trimmed = text.trim();
    if (trimmed === "") {
      if (required) revert();
      else onCommit(undefined);
      return;
    }
    const ms = Math.round(Number(trimmed));
    if (!Number.isFinite(ms) || ms <= 0) {
      revert();
      return;
    }
    onCommit(ms);
  }

  return (
    <input
      className="duration-input"
      type="number"
      min={1}
      step={10}
      value={text}
      placeholder={placeholder}
      title={title}
      onChange={(e) => setText(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") e.currentTarget.blur();
      }}
    />
  );
}
