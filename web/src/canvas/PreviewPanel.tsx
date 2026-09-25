import { useEffect, useState } from "react";
import { exportFramePngUrl } from "../api";
import type { Animation } from "../types";

interface PreviewPanelProps {
  spriteId: string;
  animation: Animation | null;
  defaultDurationMs: number;
  selectedFrameId: string | null;
  version: number;
}

// Plays the selected animation row on a loop, using each frame's own
// duration (or the sprite default). Frames are the *actual* flattened
// result via the same export.png endpoint used for real exports, so layer
// visibility/opacity — which the dotting editing canvas can't show live —
// take effect here. `version` cache-busts the URLs after anything that
// changes the composited output.
//
// Paused, it shows the selected frame (when it's in this row), which makes
// it a still composite preview of the frame being edited.
export default function PreviewPanel({
  spriteId,
  animation,
  defaultDurationMs,
  selectedFrameId,
  version,
}: PreviewPanelProps) {
  const [playing, setPlaying] = useState(true);
  const [index, setIndex] = useState(0);
  const frames = animation?.frames ?? [];

  const selectedIndex = frames.findIndex((f) => f.frameId === selectedFrameId);
  const shown = playing
    ? Math.min(index, Math.max(frames.length - 1, 0))
    : selectedIndex >= 0
      ? selectedIndex
      : 0;
  const current = frames[shown];
  const durationMs = current?.durationMs ?? defaultDurationMs;

  useEffect(() => {
    if (!playing || frames.length < 2) return;
    const t = setTimeout(() => setIndex((shown + 1) % frames.length), durationMs);
    return () => clearTimeout(t);
  }, [playing, shown, frames.length, durationMs]);

  return (
    <div className="preview-panel">
      <div className="preview-panel-header">
        <h3>Preview</h3>
        <button
          type="button"
          className="preview-panel-toggle"
          disabled={frames.length === 0}
          onClick={() => {
            if (!playing) setIndex(shown);
            setPlaying(!playing);
          }}
        >
          {playing ? "Pause" : "Play"}
        </button>
      </div>
      <div className="dotting-canvas-checkerboard preview-panel-image">
        {/* Every frame of the row is mounted at once (only one visible) so
            they're all loaded up front and playback never flickers on a
            not-yet-fetched image. */}
        {frames.map((f, i) => (
          <img
            key={f.frameId}
            src={exportFramePngUrl(spriteId, f.frameId, version)}
            alt={i === shown ? `Frame ${f.frameId}` : ""}
            style={{ visibility: i === shown ? "visible" : "hidden" }}
          />
        ))}
      </div>
      <p className="preview-panel-hint">
        {animation
          ? frames.length > 0
            ? `${animation.name} · ${shown + 1}/${frames.length} · ${durationMs}ms`
            : `${animation.name} has no frames`
          : "No animation selected"}
      </p>
    </div>
  );
}
