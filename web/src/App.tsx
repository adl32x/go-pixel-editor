import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "./api";
import DottingCanvas, {
  BrushTool,
  type DottingCanvasHandle,
} from "./canvas/DottingCanvas";
import LayerPanel from "./layers/LayerPanel";
import PaletteBar from "./palette/PaletteBar";
import PaletteSettings from "./settings/PaletteSettings";
import ClipEditor from "./sprites/ClipEditor";
import SpriteList from "./sprites/SpriteList";
import SpriteMeta from "./sprites/SpriteMeta";
import PlaybackControls from "./timeline/PlaybackControls";
import Timeline from "./timeline/Timeline";
import { spriteStore } from "./state/spriteStore";
import type { LayerProps, Settings, Sprite, SpriteSummary } from "./types";

const EMPTY_LAYERS: LayerProps[] = [];

type View = "sprites" | "settings";

export default function App() {
  const [view, setView] = useState<View>("sprites");
  const [sprites, setSprites] = useState<SpriteSummary[]>([]);
  const [spriteId, setSpriteId] = useState<string | null>(null);
  const [sprite, setSprite] = useState<Sprite | null>(null);
  const [frameId, setFrameId] = useState<string | null>(null);
  const [clipName, setClipName] = useState<string | null>(null);
  const [brushColor, setBrushColor] = useState("#000000");
  const [initLayers, setInitLayers] = useState<LayerProps[]>(EMPTY_LAYERS);
  // The project's single shared palette — every sprite draws from this same
  // list (see internal/sprite Settings), fetched once rather than per-sprite.
  const [settings, setSettings] = useState<Settings | null>(null);
  const canvasRef = useRef<DottingCanvasHandle>(null);

  const refreshSprites = useCallback(async () => {
    setSprites(await api.listSprites());
  }, []);

  useEffect(() => {
    api.getPalette().then(setSettings);
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

  // Loads a newly-selected sprite. This intentionally does NOT go through
  // selectFrame()'s imperative canvasRef.loadLayers() call: a different
  // sprite can have a different grid size and layer set than whatever
  // <DottingCanvas> is currently mounted for, and dotting's internal
  // editor does not support being repurposed for a differently-shaped
  // grid via setLayers() (it corrupts internal state — e.g.
  // this.interactionLayer becomes undefined on the next render). Instead
  // sprite/frame/initLayers are all set together here in one batch, and
  // <DottingCanvas key={sprite.id}> below remounts a fresh editor whose
  // *initial* props already match the new sprite, rather than trying to
  // swap an existing editor's data out from under it.
  useEffect(() => {
    if (!spriteId) return;
    (async () => {
      const s = await api.getSprite(spriteId);
      let firstFrameId: string | null = null;
      let layers = EMPTY_LAYERS;
      if (s.frameIds.length > 0) {
        firstFrameId = s.frameIds[0];
        layers = await api.getFrame(spriteId, firstFrameId);
      }
      setSprite(s);
      setClipName(s.clips[0]?.name ?? null);
      setFrameId(firstFrameId);
      setInitLayers(layers);
    })();
  }, [spriteId]);

  useEffect(() => {
    const clip = sprite?.clips.find((c) => c.name === clipName);
    if (clip) {
      spriteStore.setPlayback({ fps: clip.fps, loop: clip.loop, entryIndex: 0, playing: false });
    }
  }, [sprite, clipName]);

  async function handleCreateSprite(input: { name: string; width: number; height: number }) {
    const created = await api.createSprite(input);
    await api.addFrame(created.id);
    await refreshSprites();
    setSpriteId(created.id);
  }

  // Memoized: DottingCanvas's internal data-change listener + debounce
  // timer live inside a useEffect keyed on this callback's identity. An
  // unmemoized function here would get a new reference on every App
  // render (sprite list refresh, brushColor change, anything) — which
  // are frequent enough that DottingCanvas's effect cleanup would tear
  // down and rebuild the listener mid-debounce, silently cancelling
  // pending saves before the 400ms timeout ever fires.
  const handleCanvasChange = useCallback(
    async (layers: LayerProps[]) => {
      if (!spriteId || !frameId) return;
      await api.putFrame(spriteId, frameId, layers);
    },
    [spriteId, frameId],
  );

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

  // Changing the palette (PaletteSettings, via PUT /api/palette) remaps
  // every sprite's *on-disk* pixel data to the new palette — but the
  // currently-mounted <DottingCanvas>, if any, is still showing whatever it
  // loaded at mount time. Re-fetch the active frame and push it in through
  // the same imperative path selectFrame uses (identity hasn't changed, so
  // no key={sprite.id} remount happens on its own).
  async function handleSettingsChanged(updated: Settings) {
    setSettings(updated);
    if (spriteId && frameId) {
      const layers = await api.getFrame(spriteId, frameId);
      setInitLayers(layers);
      canvasRef.current?.loadLayers(layers);
    }
  }

  const clip = sprite?.clips.find((c) => c.name === clipName) ?? null;

  return (
    <div className="app">
      <aside className="app-sidebar">
        <nav className="app-nav">
          <button
            type="button"
            className={"app-nav-tab" + (view === "sprites" ? " selected" : "")}
            onClick={() => setView("sprites")}
          >
            Sprites
          </button>
          <button
            type="button"
            className={"app-nav-tab" + (view === "settings" ? " selected" : "")}
            onClick={() => setView("settings")}
          >
            Settings
          </button>
        </nav>

        {view === "sprites" && (
          <SpriteList
            sprites={sprites}
            selectedId={spriteId}
            onSelect={setSpriteId}
            onCreate={handleCreateSprite}
          />
        )}
      </aside>

      <main className="app-main">
        {view === "settings" ? (
          <PaletteSettings settings={settings} onSettingsChanged={handleSettingsChanged} />
        ) : sprite ? (
          <>
            <SpriteMeta sprite={sprite} onSave={handleSaveMeta} />

            <div className="app-workspace">
              <PaletteBar
                palette={settings?.palette ?? []}
                brushColor={brushColor}
                onSelectColor={setBrushColor}
              />

              {initLayers.length > 0 ? (
                <DottingCanvas
                  key={sprite.id}
                  ref={canvasRef}
                  initLayers={initLayers}
                  brushTool={BrushTool.DOT}
                  brushColor={brushColor}
                  onChange={handleCanvasChange}
                />
              ) : (
                <p className="app-no-frames">
                  This sprite has no frames yet — add one from the timeline below.
                </p>
              )}

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
