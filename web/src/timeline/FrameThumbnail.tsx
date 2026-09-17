import { useEffect, useRef } from "react";
import type { LayerProps } from "../types";

interface FrameThumbnailProps {
  layers: LayerProps[];
  width: number;
  height: number;
  size?: number;
}

export default function FrameThumbnail({
  layers,
  width,
  height,
  size = 48,
}: FrameThumbnailProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx || width === 0 || height === 0) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const cellW = canvas.width / width;
    const cellH = canvas.height / height;

    // Bottom layer first so later (topmost) layers paint over it.
    for (const layer of [...layers].reverse()) {
      for (const row of layer.data) {
        for (const pixel of row) {
          if (!pixel.color) continue;
          ctx.fillStyle = pixel.color;
          ctx.fillRect(
            pixel.columnIndex * cellW,
            pixel.rowIndex * cellH,
            cellW,
            cellH,
          );
        }
      }
    }
  }, [layers, width, height]);

  return (
    <canvas
      ref={canvasRef}
      width={size}
      height={size}
      className="frame-thumbnail"
    />
  );
}
