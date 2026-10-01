import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "./api";
import DottingCanvas, {
  BrushTool,
  type DottingCanvasHandle,
} from "./canvas/DottingCanvas";
import LayerPanel from "./layers/LayerPanel";
import PreviewPanel from "./canvas/PreviewPanel";
import Toolbar, { type FlipAxis, type FlipScope } from "./canvas/Toolbar";
import PaletteBar from "./palette/PaletteBar";
import BuildSettings from "./settings/BuildSettings";
import PaletteSettings from "./settings/PaletteSettings";
import SpriteList from "./sprites/SpriteList";
import SpriteMeta from "./sprites/SpriteMeta";
import FrameGrid from "./timeline/FrameGrid";
import { useHistoryCounts } from "./state/history";
import type { LayerProps, Settings, Sprite, SpriteSummary } from "./types";

const EMPTY_LAYERS: LayerProps[] = [];

type View = "sprites" | "settings";

export default function App() {
  const [view, setView] = useState<View>("sprites");
  const [sprites, setSprites] = useState<SpriteSummary[]>([]);
  const [spriteId, setSpriteId] = useState<string | null>(null);
  const [sprite, setSprite] = useState<Sprite | null>(null);
  const [frameId, setFrameId] = useState<string | null>(null);
  // The frame grid row the preview plays — follows the selected frame's
  // row, or whichever row header was clicked last.
  const [animationName, setAnimationName] = useState<string | null>(null);
  const [brushColor, setBrushColor] = useState("#000000");
  const [brushTool, setBrushTool] = useState<BrushTool>(BrushTool.DOT);
  // Which layer new strokes land on — persists across frame switches within
  // the same sprite (the layer set doesn't change when you switch frames),
  // reset to the topmost layer whenever a different sprite is opened.
  const [activeLayerId, setActiveLayerId] = useState("");
  const [initLayers, setInitLayers] = useState<LayerProps[]>(EMPTY_LAYERS);
  // The project's single shared palette — every sprite draws from this same
  // list (see internal/sprite Settings), fetched once rather than per-sprite.
  const [settings, setSettings] = useState<Settings | null>(null);
  // Bumped after anything that changes what the flattened preview (and
  // export) would look like — a pixel edit, or a layer's visibility/
  // opacity/order changing — so <PreviewPanel> can cache-bust the
  // otherwise-identical export.png URL. dotting has no opacity support and
  // (until just now) wasn't even told about visibility, so this preview is
  // the only place those two settings are actually visible live; see
  // PreviewPanel.tsx and DottingCanvas's `layers` prop.
  const [previewVersion, setPreviewVersion] = useState(0);
  const bumpPreview = useCallback(() => setPreviewVersion((v) => v + 1), []);
  // Bumped by undo/redo, which can change any frame's pixels without
  // changing any frame id — tells FrameGrid to re-fetch every thumbnail.
  const [gridReload, setGridReload] = useState(0);
  const historyCounts = useHistoryCounts(spriteId);
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

  // Keep the selected frame's row selected too, whichever way the frame got
  // selected (click, add, or a fallback after a delete).
  useEffect(() => {
    const row = sprite?.animations.find((a) => a.frames.some((f) => f.frameId === frameId));
    if (row) setAnimationName(row.name);
  }, [sprite, frameId]);

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
      setAnimationName(s.animations[0]?.name ?? null);
      setFrameId(firstFrameId);
      setInitLayers(layers);
      setActiveLayerId(s.layers[0]?.id ?? "");
    })();
  }, [spriteId]);

  async function handleCreateSprite(input: {
    name: string;
    width: number;
    height: number;
    tags?: string;
  }) {
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
      bumpPreview();
    },
    [spriteId, frameId, bumpPreview],
  );

  // Applies a sprite returned by a frame-grid mutation. If the selected
  // frame was deleted (alone or with its whole row), falls back to the first
  // frame of the selected row, then of the sprite — or to the "no frames"
  // state, which unmounts the canvas.
  async function handleGridSpriteChanged(updated: Sprite) {
    setSprite(updated);
    refreshSprites();
    if (frameId && updated.frameIds.includes(frameId)) return;
    const row = updated.animations.find((a) => a.name === animationName);
    const fallback = row?.frames[0]?.frameId ?? updated.frameIds[0];
    if (fallback) {
      await selectFrame(fallback);
    } else {
      setFrameId(null);
      setInitLayers(EMPTY_LAYERS);
      if (!row) setAnimationName(updated.animations[0]?.name ?? null);
    }
  }

  // Clicking a row header selects that row for the preview, and jumps the
  // canvas to its first frame if the current frame is in a different row.
  function handleSelectAnimation(name: string) {
    setAnimationName(name);
    const row = sprite?.animations.find((a) => a.name === name);
    if (row && row.frames.length > 0 && !row.frames.some((f) => f.frameId === frameId)) {
      selectFrame(row.frames[0].frameId);
    }
  }

  async function handleSaveMeta(patch: { name?: string; tags?: string[]; out?: string }) {
    if (!spriteId) return;
    await api.patchSprite(spriteId, patch);
    await Promise.all([refreshSprite(spriteId), refreshSprites()]);
  }

  // Resizing rewrites every frame at a new grid size. dotting can't be
  // re-shaped in place (see the sprite-switch effect above), so this swaps
  // in the resized sprite and frame data and lets the canvas key — which
  // includes the size — remount a fresh editor.
  async function handleResize(width: number, height: number, anchor: string) {
    if (!spriteId) return;
    const updated = await api.resizeSprite(spriteId, width, height, anchor);
    const layers = frameId ? await api.getFrame(spriteId, frameId) : EMPTY_LAYERS;
    setSprite(updated);
    setInitLayers(layers);
    refreshSprites();
    bumpPreview();
  }

  // Undo/redo restore a whole earlier state of the sprite server-side (see
  // internal/server/history.go), so afterwards anything may differ: pixels,
  // frames, size, layers, name. Any stroke still in the autosave debounce is
  // saved first — otherwise it would land after the undo and redo itself.
  // The canvas then shows the frame the step changed, if the current one
  // wasn't touched (or no longer exists).
  const historyBusyRef = useRef(false);
  async function handleHistoryStep(direction: "undo" | "redo") {
    if (!spriteId || historyBusyRef.current) return;
    historyBusyRef.current = true;
    try {
      await canvasRef.current?.flush();
      let step: api.HistoryStep;
      try {
        step = await (direction === "undo" ? api.undo(spriteId) : api.redo(spriteId));
      } catch {
        return; // nothing to undo/redo — the buttons just hadn't caught up
      }
      const updated = step.sprite;
      const touched = step.changedFrames.filter((id) => updated.frameIds.includes(id));
      let target = frameId && updated.frameIds.includes(frameId) ? frameId : null;
      if (touched.length > 0 && (!target || !touched.includes(target))) target = touched[0];
      if (!target) target = updated.frameIds[0] ?? null;

      const layers = target ? await api.getFrame(spriteId, target) : EMPTY_LAYERS;
      // Same size and layer set: the mounted canvas can take the data
      // directly. Otherwise its key changes and it remounts from
      // initLayers (dotting can't be reshaped in place).
      const sameShape =
        sprite !== null &&
        sprite.width === updated.width &&
        sprite.height === updated.height &&
        sprite.layers.map((l) => l.id).join() === updated.layers.map((l) => l.id).join();
      setSprite(updated);
      setFrameId(target);
      setInitLayers(layers);
      if (sameShape && layers.length > 0) canvasRef.current?.loadLayers(layers);
      if (!updated.layers.some((l) => l.id === activeLayerId)) {
        setActiveLayerId(updated.layers[0]?.id ?? "");
      }
      refreshSprites();
      bumpPreview();
      setGridReload((n) => n + 1);
    } finally {
      historyBusyRef.current = false;
    }
  }

  // Mirrors the active layer, the whole frame, or every frame of the
  // selected animation (server-side, so it's one undo step). A stroke still
  // waiting to autosave is saved first — it would otherwise land after the
  // flip and overwrite it. The same frame stays on screen, reloaded.
  async function handleFlip(axis: FlipAxis, scope: FlipScope) {
    if (!spriteId || !frameId || !sprite) return;
    await canvasRef.current?.flush();
    const row = sprite.animations.find((a) => a.frames.some((f) => f.frameId === frameId));
    const frames = scope === "animation" && row ? row.frames.map((f) => f.frameId) : [frameId];
    try {
      await api.flipFrames(spriteId, {
        axis,
        frames,
        layer: scope === "layer" ? activeLayerId : undefined,
      });
    } catch (e) {
      console.error("flip failed", e);
      return;
    }
    const layers = await api.getFrame(spriteId, frameId);
    setInitLayers(layers);
    canvasRef.current?.loadLayers(layers);
    bumpPreview();
    setGridReload((n) => n + 1);
  }

  // ⌘Z / ⇧⌘Z (Ctrl+Z / Ctrl+Y elsewhere) — except while typing in a field,
  // where the browser's own text undo should win.
  const historyStepRef = useRef(handleHistoryStep);
  historyStepRef.current = handleHistoryStep;
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
      const el = e.target as HTMLElement | null;
      if (el && (el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName))) return;
      const key = e.key.toLowerCase();
      if (key === "z") {
        e.preventDefault();
        historyStepRef.current(e.shiftKey ? "redo" : "undo");
      } else if (key === "y") {
        e.preventDefault();
        historyStepRef.current("redo");
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  async function handleLayersChange(layers: Sprite["layers"]) {
    if (!spriteId) return;
    await api.patchSprite(spriteId, { layers });
    await refreshSprite(spriteId);
    bumpPreview();
  }

  // Re-fetches the active frame and pushes it into the already-mounted
  // canvas via the same imperative path frame-switching uses. Needed
  // whenever something changes the frame's *data* out from under the open
  // canvas without changing the sprite/frame identity that key={sprite.id}
  // watches: a palette remap, or a layer being added/removed (the layer
  // *set* itself, not just pixel content — see DottingCanvas's
  // layerIdsRef for why that needs the same loadLayers path, not a diff).
  async function reloadActiveFrame() {
    if (!spriteId || !frameId) return;
    const layers = await api.getFrame(spriteId, frameId);
    setInitLayers(layers);
    canvasRef.current?.loadLayers(layers);
  }

  // Changing the palette (PaletteSettings, via PUT /api/palette) remaps
  // every sprite's *on-disk* pixel data to the new palette — but the
  // currently-mounted <DottingCanvas>, if any, is still showing whatever it
  // loaded at mount time.
  async function handleSettingsChanged(updated: Settings) {
    setSettings(updated);
    await reloadActiveFrame();
    bumpPreview();
  }

  // Adding a layer retrofits a blank block onto every frame server-side
  // (see sprite.AddLayer) — the new layer becomes the topmost one, both in
  // sprite.layers[0] and as the one the user almost certainly wants to draw
  // on immediately, so it becomes the active layer too.
  //
  // This refreshes initLayers directly (via api.getFrame) rather than going
  // through reloadActiveFrame()'s imperative canvasRef.loadLayers() call.
  // loadLayers() ultimately calls dotting's own setLayers() on the *already
  // mounted* editor, which corrupts its internal state (this.interactionLayer
  // goes undefined on the next render — confirmed live, not hypothetical)
  // whenever the *number* of layers differs from what it was initialized
  // with. <DottingCanvas>'s key includes the sprite's layer-id list (see
  // below), so updating both sprite and initLayers here — before
  // setActiveLayerId, which must not fire dotting's setCurrentLayer for a
  // layer id it doesn't know about yet either — makes React remount a fresh
  // instance with the right layers from the start instead of mutating a
  // stale one.
  async function handleAddLayer(name: string) {
    if (!spriteId || !frameId) return;
    const updated = await api.addLayer(spriteId, name);
    const layers = await api.getFrame(spriteId, frameId);
    setSprite(updated);
    setInitLayers(layers);
    setActiveLayerId(updated.layers[0]?.id ?? "");
    bumpPreview();
  }

  async function handleDeleteLayer(layerId: string) {
    if (!spriteId || !frameId) return;
    const updated = await api.deleteLayer(spriteId, layerId);
    const layers = await api.getFrame(spriteId, frameId);
    setSprite(updated);
    setInitLayers(layers);
    if (activeLayerId === layerId) {
      setActiveLayerId(updated.layers[0]?.id ?? "");
    }
    bumpPreview();
  }

  const animation = sprite?.animations.find((a) => a.name === animationName) ?? null;

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
          <>
            <PaletteSettings settings={settings} onSettingsChanged={handleSettingsChanged} />
            <BuildSettings settings={settings} onSettingsChanged={setSettings} />
          </>
        ) : sprite ? (
          <>
            {/* SpriteMeta's name/tags fields are local useState seeded once
                from props — without a key forcing a remount on sprite
                switch, they'd keep showing the *previous* sprite's values
                until the user manually edits them, risking a save that
                silently renames the wrong sprite. */}
            <SpriteMeta
              key={sprite.id}
              sprite={sprite}
              onSave={handleSaveMeta}
              onResize={handleResize}
            />

            <div className="app-workspace">
              <PaletteBar
                palette={settings?.palette ?? []}
                brushColor={brushColor}
                onSelectColor={setBrushColor}
              />

              <Toolbar
                tool={brushTool}
                onSelectTool={setBrushTool}
                canUndo={historyCounts.undo > 0}
                canRedo={historyCounts.redo > 0}
                onUndo={() => handleHistoryStep("undo")}
                onRedo={() => handleHistoryStep("redo")}
                onFlip={handleFlip}
              />

              {initLayers.length > 0 ? (
                <DottingCanvas
                  // dotting's setLayers()/loadLayers() corrupts its internal
                  // editor state (this.interactionLayer goes undefined on
                  // the next render — a confirmed bug, not just a risk) when
                  // called on a mounted instance whose *layer count* differs
                  // from what it was initialized with, not only when the
                  // grid dimensions differ. Layer add/delete changes the
                  // layer count, so — like a sprite switch — it needs a full
                  // remount with fresh initLayers rather than an imperative
                  // update; see handleAddLayer/handleDeleteLayer, which
                  // refresh initLayers themselves instead of going through
                  // reloadActiveFrame()'s loadLayers() call for this reason.
                  key={
                    sprite.id +
                    ":" +
                    sprite.width +
                    "x" +
                    sprite.height +
                    ":" +
                    sprite.layers.map((l) => l.id).join(",")
                  }
                  ref={canvasRef}
                  initLayers={initLayers}
                  brushTool={brushTool}
                  brushColor={brushColor}
                  activeLayerId={activeLayerId}
                  layers={sprite.layers}
                  onChange={handleCanvasChange}
                />
              ) : (
                <p className="app-no-frames">
                  This sprite has no frames yet — add one from the frame grid below.
                </p>
              )}

              <LayerPanel
                layers={sprite.layers}
                activeLayerId={activeLayerId}
                onSelectLayer={setActiveLayerId}
                onChange={handleLayersChange}
                onAddLayer={handleAddLayer}
                onDeleteLayer={handleDeleteLayer}
              />

              <PreviewPanel
                // Restart playback from the row's first frame on a row switch.
                key={sprite.id + "/" + (animation?.name ?? "")}
                spriteId={sprite.id}
                width={sprite.width}
                height={sprite.height}
                animation={animation}
                defaultDurationMs={sprite.durationMs}
                selectedFrameId={frameId}
                version={previewVersion}
              />
            </div>

            <FrameGrid
              sprite={sprite}
              selectedFrameId={frameId}
              selectedAnimation={animationName}
              version={previewVersion}
              reloadKey={gridReload}
              onSelectFrame={selectFrame}
              onSelectAnimation={handleSelectAnimation}
              onSpriteChanged={handleGridSpriteChanged}
            />
          </>
        ) : (
          <p>Select or create a sprite to begin.</p>
        )}
      </main>
    </div>
  );
}
