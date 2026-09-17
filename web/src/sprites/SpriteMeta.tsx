import { useState } from "react";
import type { Sprite } from "../types";

interface SpriteMetaProps {
  sprite: Sprite;
  onSave: (patch: { name?: string; tags?: string[] }) => void;
}

export default function SpriteMeta({ sprite, onSave }: SpriteMetaProps) {
  const [name, setName] = useState(sprite.name);
  const [tags, setTags] = useState(sprite.tags.join(", "));

  function handleBlurSave() {
    onSave({
      name,
      tags: tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
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
      <span className="sprite-meta-dims">
        {sprite.width}x{sprite.height}
      </span>
    </div>
  );
}
