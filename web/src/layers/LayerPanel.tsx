import { useState } from "react";
import type { LayerDef } from "../types";

interface LayerPanelProps {
  layers: LayerDef[];
  activeLayerId: string;
  onSelectLayer: (id: string) => void;
  onChange: (layers: LayerDef[]) => void;
  onAddLayer: (name: string) => void;
  onDeleteLayer: (id: string) => void;
}

export default function LayerPanel({
  layers,
  activeLayerId,
  onSelectLayer,
  onChange,
  onAddLayer,
  onDeleteLayer,
}: LayerPanelProps) {
  const [newName, setNewName] = useState("");

  function update(id: string, patch: Partial<LayerDef>) {
    onChange(layers.map((l) => (l.id === id ? { ...l, ...patch } : l)));
  }

  // Reordering never touches frame files (see #0028) — it's just a
  // whole-array-replace patch through the same onChange as rename/opacity/
  // visibility, exactly like the sprite-metadata pattern used elsewhere.
  function move(index: number, dir: -1 | 1) {
    const target = index + dir;
    if (target < 0 || target >= layers.length) return;
    const next = [...layers];
    [next[index], next[target]] = [next[target], next[index]];
    onChange(next);
  }

  function handleAdd() {
    onAddLayer(newName.trim() || "Layer");
    setNewName("");
  }

  return (
    <div className="layer-panel">
      <h3>Layers</h3>
      <ul>
        {layers.map((layer, i) => (
          <li
            key={layer.id}
            className={"layer-row" + (layer.id === activeLayerId ? " selected" : "")}
            onClick={() => onSelectLayer(layer.id)}
          >
            <input
              type="checkbox"
              checked={layer.visible}
              onChange={(e) => {
                e.stopPropagation();
                update(layer.id, { visible: e.target.checked });
              }}
              onClick={(e) => e.stopPropagation()}
              title="visible"
            />
            <input
              className="layer-name"
              value={layer.name}
              onChange={(e) => update(layer.id, { name: e.target.value })}
              onClick={(e) => e.stopPropagation()}
              title="rename"
            />
            <input
              type="range"
              min={0}
              max={1}
              step={0.05}
              value={layer.opacity}
              onChange={(e) => update(layer.id, { opacity: Number(e.target.value) })}
              onClick={(e) => e.stopPropagation()}
              title="opacity"
            />
            <div className="layer-row-actions">
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  move(i, -1);
                }}
                disabled={i === 0}
                title="move up"
              >
                &uarr;
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  move(i, 1);
                }}
                disabled={i === layers.length - 1}
                title="move down"
              >
                &darr;
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteLayer(layer.id);
                }}
                disabled={layers.length <= 1}
                title={layers.length <= 1 ? "a sprite needs at least one layer" : "delete layer"}
              >
                &times;
              </button>
            </div>
          </li>
        ))}
      </ul>
      <div className="layer-create">
        <input
          placeholder="new layer name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
        />
        <button type="button" onClick={handleAdd}>
          + Layer
        </button>
      </div>
    </div>
  );
}
