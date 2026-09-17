import type { LayerDef } from "../types";

interface LayerPanelProps {
  layers: LayerDef[];
  onChange: (layers: LayerDef[]) => void;
}

export default function LayerPanel({ layers, onChange }: LayerPanelProps) {
  function update(id: string, patch: Partial<LayerDef>) {
    onChange(layers.map((l) => (l.id === id ? { ...l, ...patch } : l)));
  }

  return (
    <div className="layer-panel">
      <h3>Layers</h3>
      <ul>
        {layers.map((layer) => (
          <li key={layer.id} className="layer-row">
            <input
              type="checkbox"
              checked={layer.visible}
              onChange={(e) => update(layer.id, { visible: e.target.checked })}
              title="visible"
            />
            <span className="layer-name">{layer.name}</span>
            <input
              type="range"
              min={0}
              max={1}
              step={0.05}
              value={layer.opacity}
              onChange={(e) => update(layer.id, { opacity: Number(e.target.value) })}
              title="opacity"
            />
          </li>
        ))}
      </ul>
    </div>
  );
}
