import { useEffect, useRef } from "react";
import { spriteStore, useSpriteStore } from "../state/spriteStore";
import type { Clip, LoopMode } from "../types";

interface PlaybackControlsProps {
  clip: Clip | null;
  onFrame: (frameId: string) => void;
}

function nextIndex(index: number, length: number, loop: LoopMode, dir: 1 | -1): {
  index: number;
  dir: 1 | -1;
} {
  const proposed = index + dir;
  if (proposed >= 0 && proposed < length) return { index: proposed, dir };

  switch (loop) {
    case "none":
      return { index: Math.min(Math.max(index, 0), length - 1), dir };
    case "forward":
      return { index: dir === 1 ? 0 : length - 1, dir };
    case "pingpong": {
      const flipped = dir === 1 ? -1 : 1;
      return { index: index + flipped, dir: flipped };
    }
  }
}

export default function PlaybackControls({ clip, onFrame }: PlaybackControlsProps) {
  const { playback } = useSpriteStore();
  const dirRef = useRef<1 | -1>(1);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    if (!playback.playing || !clip || clip.entries.length === 0) return;

    const tick = () => {
      const state = spriteStore.getState().playback;
      const entry = clip.entries[state.entryIndex];
      const durationMs = entry?.durationMs ?? 1000 / Math.max(state.fps, 0.1);
      onFrame(entry.frameId);

      const { index, dir } = nextIndex(
        state.entryIndex,
        clip.entries.length,
        state.loop,
        dirRef.current,
      );
      dirRef.current = dir;

      if (state.loop === "none" && index === state.entryIndex) {
        spriteStore.setPlayback({ playing: false });
        return;
      }
      spriteStore.setPlayback({ entryIndex: index });
      timerRef.current = setTimeout(tick, durationMs);
    };

    timerRef.current = setTimeout(tick, 0);
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playback.playing, clip]);

  if (!clip) {
    return <div className="playback-controls playback-controls-empty">No clip selected</div>;
  }

  return (
    <div className="playback-controls">
      <button
        type="button"
        onClick={() => spriteStore.setPlayback({ playing: !playback.playing })}
      >
        {playback.playing ? "Pause" : "Play"}
      </button>
      <label>
        FPS
        <input
          type="number"
          min={1}
          max={60}
          value={playback.fps}
          onChange={(e) => spriteStore.setPlayback({ fps: Number(e.target.value) })}
        />
      </label>
      <label>
        Loop
        <select
          value={playback.loop}
          onChange={(e) =>
            spriteStore.setPlayback({ loop: e.target.value as LoopMode })
          }
        >
          <option value="none">none</option>
          <option value="forward">forward</option>
          <option value="pingpong">pingpong</option>
        </select>
      </label>
      <input
        type="range"
        min={0}
        max={Math.max(clip.entries.length - 1, 0)}
        value={playback.entryIndex}
        onChange={(e) => {
          const index = Number(e.target.value);
          spriteStore.setPlayback({ entryIndex: index });
          const entry = clip.entries[index];
          if (entry) onFrame(entry.frameId);
        }}
      />
    </div>
  );
}
