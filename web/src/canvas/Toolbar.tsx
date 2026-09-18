import { BrushTool } from "./DottingCanvas";

interface ToolbarProps {
  tool: BrushTool;
  onSelectTool: (tool: BrushTool) => void;
}

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

export default function Toolbar({ tool, onSelectTool }: ToolbarProps) {
  return (
    <div className="toolbar">
      <h3>Tools</h3>
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
    </div>
  );
}
