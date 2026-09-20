import { exportFramePngUrl } from "../api";

interface PreviewPanelProps {
  spriteId: string;
  frameId: string;
  version: number;
}

// The live <DottingCanvas> can't preview layer visibility or opacity —
// dotting has no opacity concept at all, and (independent of that) has its
// own separate render pipeline from the Go backend's export compositor.
// This renders the *actual* flattened result via the same export.png
// endpoint used for real exports, so visibility/opacity always have
// somewhere they visibly take effect even though the editing canvas can't
// show them live. `version` cache-busts the URL after anything that
// changes the composited output but not the URL's spriteId/frameId.
export default function PreviewPanel({ spriteId, frameId, version }: PreviewPanelProps) {
  return (
    <div className="preview-panel">
      <h3>Preview</h3>
      <div className="dotting-canvas-checkerboard preview-panel-image">
        <img
          key={spriteId + "/" + frameId}
          src={exportFramePngUrl(spriteId, frameId, version)}
          alt="Flattened preview"
        />
      </div>
      <p className="preview-panel-hint">
        The true composited result — honors layer visibility and opacity,
        neither of which the editing canvas above can show live.
      </p>
    </div>
  );
}
