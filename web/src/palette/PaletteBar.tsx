import type { PaletteEntry } from "../types";

interface PaletteBarProps {
  palette: PaletteEntry[];
  brushColor: string;
  onSelectColor: (color: string) => void;
}

export default function PaletteBar({ palette, brushColor, onSelectColor }: PaletteBarProps) {
  return (
    <div className="palette-bar">
      <h3>Palette</h3>
      <div className="palette-swatches">
        {palette.map((entry) => (
          <button
            key={entry.char}
            type="button"
            className={"palette-swatch" + (entry.color === brushColor ? " selected" : "")}
            style={{ backgroundColor: entry.color }}
            title={entry.color}
            onClick={() => onSelectColor(entry.color)}
          />
        ))}
      </div>
      <input
        type="color"
        value={/^#[0-9a-fA-F]{6}$/.test(brushColor) ? brushColor : "#000000"}
        onChange={(e) => onSelectColor(e.target.value)}
      />
    </div>
  );
}
