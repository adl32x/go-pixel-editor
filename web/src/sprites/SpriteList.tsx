import { useState } from "react";
import type { SpriteSummary } from "../types";

interface SpriteListProps {
  sprites: SpriteSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onCreate: (input: { name: string; width: number; height: number }) => void;
}

export default function SpriteList({
  sprites,
  selectedId,
  onSelect,
  onCreate,
}: SpriteListProps) {
  const [name, setName] = useState("");
  const [width, setWidth] = useState(16);
  const [height, setHeight] = useState(16);

  function handleCreate() {
    if (!name.trim()) return;
    onCreate({ name: name.trim(), width, height });
    setName("");
  }

  return (
    <div className="sprite-list">
      <h3>Sprites</h3>
      <ul>
        {sprites.map((s) => (
          <li key={s.id}>
            <button
              type="button"
              className={"sprite-list-item" + (s.id === selectedId ? " selected" : "")}
              onClick={() => onSelect(s.id)}
            >
              {s.id} — {s.name} ({s.width}x{s.height})
            </button>
          </li>
        ))}
      </ul>
      <div className="sprite-create">
        <div className="sprite-create-fields">
          <input
            placeholder="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            type="number"
            min={1}
            value={width}
            onChange={(e) => setWidth(Number(e.target.value))}
          />
          <input
            type="number"
            min={1}
            value={height}
            onChange={(e) => setHeight(Number(e.target.value))}
          />
        </div>
        <button type="button" className="sprite-create-submit" onClick={handleCreate}>
          New sprite
        </button>
      </div>
    </div>
  );
}
