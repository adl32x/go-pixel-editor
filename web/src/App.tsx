import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "./api";
import DottingCanvas, {
  BrushTool,
  type DottingCanvasHandle,
} from "./canvas/DottingCanvas";
import LayerPanel from "./layers/LayerPanel";
import PaletteBar from "./palette/PaletteBar";
import ClipEditor from "./sprites/ClipEditor";
import SpriteList from "./sprites/SpriteList";
import SpriteMeta from "./sprites/SpriteMeta";
import PlaybackControls from "./timeline/PlaybackControls";
import Timeline from "./timeline/Timeline";
import { spriteStore } from "./state/spriteStore";
import type { LayerProps, Sprite, SpriteSummary } from "./types";

const EMPTY_LAYERS: LayerProps[] = [];

export default function App() {
  const [sprites, setSprites] = useState<SpriteSummary[]>([]);
  const [spriteId, setSpriteId] = useState<string | null>(null);
  const [sprite, setSprite] = useState<Sprite | null>(null);
  const [frameId, setFrameId] = useState<string | null>(null);
  const [clipName, setClipName] = useState<string | null>(null);
  const [brushColor, setBrushColor] = useState("#000000");
  const [initLayers, setInitLayers] = useState<LayerProps[]>(EMPTY_LAYERS);
  const canvasRef = useRef<DottingCanvasHandle>(null);

  const refreshSprites = useCallback(async () => {
    setSprites(await api.listSprites());
  }, []);

  const refreshSprite = useCallback(async (id: string) => {
    const s = await api.getSprite(id);
    setSprite(s);
    return s;
  }, []);

  useEffect(() => {
    refreshSprites();
  }, [refreshSprites]);

  const selectFrame = useCallback(async (id: string) => {
    if (!spriteId) return;
    const layers = await api.getFrame(spriteId, id);
    setFrameId(id);
    setInitLayers(layers);
    canvasRef.current?.loadLayers(layers);
  }, [spriteId]);

  useEffect(() => {
    if (!spriteId) return;
    (async () => {
      const s = await refreshSprite(spriteId);
      setClipName(s.clips[0]?.name ?? null);
      if (s.frameIds.length > 0) {
        await selectFrame(s.frameIds[0]);
      } else {
        setFrameId(null);
        setInitLayers(EMPTY_LAYERS);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [spriteId, refreshSprite]);

  useEffect(() => {
    const clip = sprite?.clips.find((c) => c.name === clipName);
    if (clip) {
      spriteStore.setPlayback({ fps: clip.fps, loop: clip.loop, entryIndex: 0, playing: false });
    }
  }, [sprite, clipName]);

  async function handleCreateSprite(input: { name: string; width: number; height: number }) {
    const created = await api.createSprite(input);
    await refreshSprites();
    setSpriteId(created.id);
  }

  async function handleCanvasChange(layers: LayerProps[]) {
    if (!spriteId || !frameId) return;
    await api.putFrame(spriteId, frameId, layers);
  }

  async function handleFramesChanged() {
    if (!spriteId) return;
    await refreshSprite(spriteId);
  }

  async function handleClipsChanged() {
    if (!spriteId) return;
    await refreshSprite(spriteId);
  }

  async function handleSaveMeta(patch: { name?: string; tags?: string[] }) {
    if (!spriteId) return;
    await api.patchSprite(spriteId, patch);
    await Promise.all([refreshSprite(spriteId), refreshSprites()]);
  }

  async function handleLayersChange(layers: Sprite["layers"]) {
    if (!spriteId) return;
    await api.patchSprite(spriteId, { layers });
    await refreshSprite(spriteId);
  }

  const clip = sprite?.clips.find((c) => c.name === clipName) ?? null;

  return (
    <div className="app">
      <aside className="app-sidebar">
        <SpriteList
          sprites={sprites}
          selectedId={spriteId}
          onSelect={setSpriteId}
          onCreate={handleCreateSprite}
        />
      </aside>

      <main className="app-main">
        {sprite ? (
          <>
            <SpriteMeta sprite={sprite} onSave={handleSaveMeta} />

            <div className="app-workspace">
              <PaletteBar
                palette={sprite.palette}
                brushColor={brushColor}
                onSelectColor={setBrushColor}
              />

              <DottingCanvas
                ref={canvasRef}
                initLayers={initLayers}
                brushTool={BrushTool.DOT}
                brushColor={brushColor}
                onChange={handleCanvasChange}
              />

              <LayerPanel layers={sprite.layers} onChange={handleLayersChange} />
            </div>

            <Timeline
              spriteId={sprite.id}
              width={sprite.width}
              height={sprite.height}
              frameIds={sprite.frameIds}
              selectedFrameId={frameId}
              onSelectFrame={selectFrame}
              onFramesChanged={handleFramesChanged}
            />

            <PlaybackControls clip={clip} onFrame={selectFrame} />

            <ClipEditor
              spriteId={sprite.id}
              clips={sprite.clips}
              selectedClipName={clipName}
              selectedFrameId={frameId}
              onSelectClip={setClipName}
              onClipsChanged={handleClipsChanged}
            />
          </>
        ) : (
          <p>Select or create a sprite to begin.</p>
        )}
      </main>
    </div>
  );
}
