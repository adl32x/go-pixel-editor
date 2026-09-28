import { useState } from "react";

// Same names, same 3x3 order as sprite.Anchors on the Go side.
const ANCHORS = [
  "top-left", "top", "top-right",
  "left", "center", "right",
  "bottom-left", "bottom", "bottom-right",
] as const;

const ARROWS: Record<(typeof ANCHORS)[number], string> = {
  "top-left": "↖", top: "↑", "top-right": "↗",
  left: "←", center: "•", right: "→",
  "bottom-left": "↙", bottom: "↓", "bottom-right": "↘",
};

interface ResizeControlProps {
  width: number;
  height: number;
  onResize: (width: number, height: number, anchor: string) => Promise<void>;
}

// The sprite's canvas size, with an inline form to change it. Growing pads
// with transparent pixels; shrinking crops (after a confirm). The anchor is
// the part of the current canvas that stays in place.
export default function ResizeControl({ width, height, onResize }: ResizeControlProps) {
  const [open, setOpen] = useState(false);
  const [w, setW] = useState(String(width));
  const [h, setH] = useState(String(height));
  const [anchor, setAnchor] = useState<string>("center");
  const [error, setError] = useState<string | null>(null);

  function start() {
    setW(String(width));
    setH(String(height));
    setError(null);
    setOpen(true);
  }

  async function apply() {
    const nw = Math.round(Number(w));
    const nh = Math.round(Number(h));
    if (!(nw >= 1 && nh >= 1)) {
      setError("width and height must be at least 1");
      return;
    }
    if (nw === width && nh === height) {
      setOpen(false);
      return;
    }
    if (
      (nw < width || nh < height) &&
      !confirm(`Shrink to ${nw}x${nh}? Pixels outside the new canvas are cut off in every frame.`)
    ) {
      return;
    }
    try {
      await onResize(nw, nh, anchor);
      setOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  if (!open) {
    return (
      <span className="resize-control">
        <span className="sprite-meta-dims">
          {width}x{height}
        </span>
        <button type="button" className="resize-control-open" onClick={start}>
          Resize
        </button>
      </span>
    );
  }

  return (
    <span className="resize-control open">
      <input
        type="number"
        min={1}
        aria-label="Width"
        value={w}
        onChange={(e) => setW(e.target.value)}
      />
      ×
      <input
        type="number"
        min={1}
        aria-label="Height"
        value={h}
        onChange={(e) => setH(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && apply()}
      />
      <span className="resize-anchor" role="radiogroup" aria-label="Anchor">
        {ANCHORS.map((a) => (
          <button
            key={a}
            type="button"
            role="radio"
            aria-checked={a === anchor}
            title={`Keep ${a} in place`}
            className={a === anchor ? "selected" : ""}
            onClick={() => setAnchor(a)}
          >
            {ARROWS[a]}
          </button>
        ))}
      </span>
      <button type="button" onClick={apply}>
        Apply
      </button>
      <button type="button" onClick={() => setOpen(false)}>
        Cancel
      </button>
      {error && <span className="frame-grid-error">{error}</span>}
    </span>
  );
}
