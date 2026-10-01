import { useState } from "react";
import { BrushTool } from "./DottingCanvas";

interface ToolbarProps {
  tool: BrushTool;
  onSelectTool: (tool: BrushTool) => void;
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  onFlip: (axis: FlipAxis, scope: FlipScope) => void;
}

export type FlipAxis = "horizontal" | "vertical";
// What a flip applies to: the active layer of this frame, every layer of
// this frame, or every frame of the selected animation row.
export type FlipScope = "layer" | "frame" | "animation";

const FLIP_SCOPES: Array<{ scope: FlipScope; label: string; title: string }> = [
  { scope: "layer", label: "Layer", title: "Flip the active layer of this frame" },
  { scope: "frame", label: "Frame", title: "Flip every layer of this frame" },
  { scope: "animation", label: "Anim", title: "Flip every frame of the selected animation" },
];

// Shown in tooltips; the shortcuts themselves are handled in App.
const MOD = /Mac|iPhone|iPad/.test(navigator.platform) ? "⌘" : "Ctrl+";

// BrushTool.NONE is dotting's "nothing selected, just pan" mode — not a
// drawing tool a user would deliberately pick from a toolbar, and panning is
// already reachable via middle-click/scroll regardless of the active tool
// (see DottingCanvas's isPanZoomable), so it's left out here on purpose
// rather than by oversight.
const TOOLS: Array<{ tool: BrushTool; label: string }> = [
  { tool: BrushTool.DOT, label: "Pen" },
  { tool: BrushTool.ERASER, label: "Eraser" },
  { tool: BrushTool.PAINT_BUCKET, label: "Fill" },
  { tool: BrushTool.SELECT, label: "Select" },
  { tool: BrushTool.LINE, label: "Line" },
  { tool: BrushTool.RECTANGLE, label: "Rect" },
  { tool: BrushTool.RECTANGLE_FILLED, label: "Rect (filled)" },
  { tool: BrushTool.ELLIPSE, label: "Ellipse" },
  { tool: BrushTool.ELLIPSE_FILLED, label: "Ellipse (filled)" },
];

export default function Toolbar({
  tool,
  onSelectTool,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  onFlip,
}: ToolbarProps) {
  const [flipScope, setFlipScope] = useState<FlipScope>("frame");
  const scopeTitle = FLIP_SCOPES.find((s) => s.scope === flipScope)!.title.replace("Flip ", "");

  return (
    <div className="toolbar">
      <h3>Tools</h3>
      <div className="toolbar-history">
        <button type="button" disabled={!canUndo} title={`Undo (${MOD}Z)`} onClick={onUndo}>
          Undo
        </button>
        <button
          type="button"
          disabled={!canRedo}
          title={`Redo (${MOD === "⌘" ? "⇧⌘Z" : "Ctrl+Y"})`}
          onClick={onRedo}
        >
          Redo
        </button>
      </div>
      <div className="toolbar-buttons">
        {TOOLS.map((t) => (
          <button
            key={t.tool}
            type="button"
            className={"toolbar-button" + (t.tool === tool ? " selected" : "")}
            onClick={() => onSelectTool(t.tool)}
          >
            {t.label}
          </button>
        ))}
      </div>
      <h3 className="toolbar-section">Flip</h3>
      <div className="toolbar-flip">
        <button
          type="button"
          title={`Flip horizontally (left ↔ right): ${scopeTitle}`}
          onClick={() => onFlip("horizontal", flipScope)}
        >
          ↔
        </button>
        <button
          type="button"
          title={`Flip vertically (top ↕ bottom): ${scopeTitle}`}
          onClick={() => onFlip("vertical", flipScope)}
        >
          ↕
        </button>
      </div>
      <div className="toolbar-flip-scope" role="radiogroup" aria-label="Flip applies to">
        {FLIP_SCOPES.map((s) => (
          <button
            key={s.scope}
            type="button"
            role="radio"
            aria-checked={s.scope === flipScope}
            title={s.title}
            className={s.scope === flipScope ? "selected" : ""}
            onClick={() => setFlipScope(s.scope)}
          >
            {s.label}
          </button>
        ))}
      </div>
    </div>
  );
}
