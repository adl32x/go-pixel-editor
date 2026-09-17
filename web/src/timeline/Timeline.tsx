import { useEffect, useState } from "react";
import * as api from "../api";
import type { LayerProps } from "../types";
import FrameThumbnail from "./FrameThumbnail";

interface TimelineProps {
  spriteId: string;
  width: number;
  height: number;
  frameIds: string[];
  selectedFrameId: string | null;
  onSelectFrame: (frameId: string) => void;
  onFramesChanged: () => void;
}

export default function Timeline({
  spriteId,
  width,
  height,
  frameIds,
  selectedFrameId,
  onSelectFrame,
  onFramesChanged,
}: TimelineProps) {
  const [thumbs, setThumbs] = useState<Record<string, LayerProps[]>>({});

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const entries = await Promise.all(
        frameIds.map(async (id) => {
          try {
            return [id, await api.getFrame(spriteId, id)] as const;
          } catch {
            return [id, []] as const;
          }
        }),
      );
      if (!cancelled) {
        setThumbs(Object.fromEntries(entries));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [spriteId, frameIds]);

  async function handleAddFrame() {
    const { frameId } = await api.addFrame(spriteId);
    onFramesChanged();
    onSelectFrame(frameId);
  }

  async function handleDeleteFrame(frameId: string) {
    try {
      await api.deleteFrame(spriteId, frameId);
    } catch {
      if (confirm(`Frame ${frameId} is used by a clip. Delete anyway?`)) {
        await api.deleteFrame(spriteId, frameId, true);
      } else {
        return;
      }
    }
    onFramesChanged();
  }

  return (
    <div className="timeline">
      <div className="timeline-strip">
        {frameIds.map((frameId) => (
          <button
            key={frameId}
            type="button"
            className={
              "timeline-frame" + (frameId === selectedFrameId ? " selected" : "")
            }
            onClick={() => onSelectFrame(frameId)}
          >
            <FrameThumbnail layers={thumbs[frameId] ?? []} width={width} height={height} />
            <span className="timeline-frame-label">{frameId}</span>
            <span
              className="timeline-frame-delete"
              role="button"
              tabIndex={0}
              onClick={(e) => {
                e.stopPropagation();
                handleDeleteFrame(frameId);
              }}
            >
              &times;
            </span>
          </button>
        ))}
        <button type="button" className="timeline-add" onClick={handleAddFrame}>
          + Frame
        </button>
      </div>
    </div>
  );
}
