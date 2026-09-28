import { useState } from "react";
import type { Sprite } from "../types";
import ResizeControl from "./ResizeControl";

interface SpriteMetaProps {
  sprite: Sprite;
  onSave: (patch: { name?: string; tags?: string[]; out?: string }) => Promise<void>;
  onResize: (width: number, height: number, anchor: string) => Promise<void>;
}

export default function SpriteMeta({ sprite, onSave, onResize }: SpriteMetaProps) {
  const [name, setName] = useState(sprite.name);
  const [tags, setTags] = useState(sprite.tags.join(", "));
  const [out, setOut] = useState(sprite.out);
  const [error, setError] = useState<string | null>(null);

  async function handleBlurSave() {
    try {
      await save();
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  function save() {
    return onSave({
      name,
      tags: tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
      out,
    });
  }

  return (
    <div className="sprite-meta">
      <input value={name} onChange={(e) => setName(e.target.value)} onBlur={handleBlurSave} />
      <input
        value={tags}
        placeholder="tags, comma, separated"
        onChange={(e) => setTags(e.target.value)}
        onBlur={handleBlurSave}
      />
      <input
        value={out}
        placeholder="output subfolder"
        title="Subfolder of the build folder (Settings) this sprite builds into, e.g. characters"
        onChange={(e) => setOut(e.target.value)}
        onBlur={handleBlurSave}
      />
      <ResizeControl width={sprite.width} height={sprite.height} onResize={onResize} />
      {error && <span className="frame-grid-error">{error}</span>}
    </div>
  );
}
