import { useEffect, useMemo, useState } from "react";
import type { SpriteSummary } from "../types";

interface SpriteListProps {
  sprites: SpriteSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onCreate: (input: { name: string; width: number; height: number; tags?: string }) => void;
}

// Stands for "sprites with no tags" in the filter — not a real tag (tags
// come from sprite.md, where this can't collide: it's not a word).
const UNTAGGED = "\u0000untagged";

const FILTER_KEY = "pixel.spriteTagFilter";

function loadFilter(): string[] {
  try {
    const raw = localStorage.getItem(FILTER_KEY);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((t) => typeof t === "string") : [];
  } catch {
    return [];
  }
}

function saveFilter(tags: string[]) {
  try {
    localStorage.setItem(FILTER_KEY, JSON.stringify(tags));
  } catch {
    // private window / blocked storage — the filter just won't persist
  }
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
  const [filter, setFilter] = useState<string[]>(loadFilter);

  // Every tag in use, with how many sprites have it, most used first.
  const tagCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const s of sprites) {
      for (const t of s.tags) counts.set(t, (counts.get(t) ?? 0) + 1);
    }
    return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  }, [sprites]);
  const untaggedCount = sprites.filter((s) => s.tags.length === 0).length;
  const maxCount = Math.max(1, ...tagCounts.map(([, n]) => n));

  // Drop tags from the filter once no sprite has them any more (renamed
  // or removed), so the list can't silently stay filtered to nothing.
  useEffect(() => {
    if (sprites.length === 0) return;
    const live = new Set(tagCounts.map(([t]) => t));
    if (untaggedCount > 0) live.add(UNTAGGED);
    const kept = filter.filter((t) => live.has(t));
    if (kept.length !== filter.length) {
      setFilter(kept);
      saveFilter(kept);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tagCounts, untaggedCount]);

  // Functional update, so two quick clicks can't both start from the same
  // stale filter.
  function toggle(tag: string) {
    setFilter((current) => {
      const next = current.includes(tag) ? current.filter((t) => t !== tag) : [...current, tag];
      saveFilter(next);
      return next;
    });
  }

  // A sprite shows if it has *any* selected tag — tags like "monsters" and
  // "icons" are categories, so requiring all of them would show nothing.
  const visible =
    filter.length === 0
      ? sprites
      : sprites.filter(
          (s) =>
            (filter.includes(UNTAGGED) && s.tags.length === 0) ||
            s.tags.some((t) => filter.includes(t)),
        );
  const filterTags = filter.filter((t) => t !== UNTAGGED);

  function handleCreate() {
    if (!name.trim()) return;
    // A sprite made while filtering gets the filter's tags, so it doesn't
    // vanish from the list the moment it's created.
    onCreate({ name: name.trim(), width, height, tags: filterTags.join(", ") || undefined });
    setName("");
  }

  // Classic tag-cloud sizing: more-used tags are a bit bigger.
  const chipSize = (count: number) => 11 + Math.round((count / maxCount) * 3);

  return (
    <div className="sprite-list">
      <h3>Sprites</h3>
      {(tagCounts.length > 0 || untaggedCount > 0) && (
        <div className="tag-cloud" role="group" aria-label="Filter sprites by tag">
          {tagCounts.map(([tag, count]) => (
            <button
              key={tag}
              type="button"
              className={"tag-chip" + (filter.includes(tag) ? " selected" : "")}
              aria-pressed={filter.includes(tag)}
              style={{ fontSize: chipSize(count) }}
              onClick={() => toggle(tag)}
            >
              {tag} <span className="tag-chip-count">{count}</span>
            </button>
          ))}
          {untaggedCount > 0 && tagCounts.length > 0 && (
            <button
              type="button"
              className={"tag-chip untagged" + (filter.includes(UNTAGGED) ? " selected" : "")}
              aria-pressed={filter.includes(UNTAGGED)}
              title="Sprites without any tags"
              onClick={() => toggle(UNTAGGED)}
            >
              untagged <span className="tag-chip-count">{untaggedCount}</span>
            </button>
          )}
          {filter.length > 0 && (
            <button
              type="button"
              className="tag-cloud-clear"
              onClick={() => {
                setFilter([]);
                saveFilter([]);
              }}
            >
              Show all
            </button>
          )}
        </div>
      )}
      <ul>
        {visible.map((s) => (
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
      {filter.length > 0 && (
        <p className="tag-cloud-hint">
          Showing {visible.length} of {sprites.length}
          {filterTags.length > 0 && <> · new sprites get {filterTags.join(", ")}</>}
        </p>
      )}
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
