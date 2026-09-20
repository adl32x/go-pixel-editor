import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import {
  BrushTool,
  Dotting,
  type CanvasDataChangeParams,
  type DottingRef,
  type LayerProps,
  type PixelModifyItem,
} from "dotting";
import type { LayerDef } from "../types";

export interface DottingCanvasHandle {
  loadLayers: (layers: LayerProps[]) => void;
  getLayers: () => LayerProps[];
}

interface DottingCanvasProps {
  width?: number;
  height?: number;
  initLayers: LayerProps[];
  brushTool: BrushTool;
  brushColor: string;
  // Which layer new strokes land on. dotting has its own internal "current
  // layer" state but no UI of its own to change it (we have our own
  // LayerPanel for that) — this prop drives dotting's setCurrentLayer ref
  // method imperatively, the same way loadLayers drives setLayers.
  activeLayerId: string;
  // Sprite-level layer metadata (visibility, in particular) — dotting's own
  // LayerProps only carries {id, data}, with no visibility flag, so this is
  // a separate prop rather than folded into initLayers. Drives dotting's
  // showLayer/hideLayer ref methods the same imperative way activeLayerId
  // drives setCurrentLayer, since dotting has no reactive prop for this
  // either. Note there is no equivalent for `opacity` — dotting has no
  // opacity concept at all (confirmed: zero references anywhere in its
  // source), so it can only ever be applied to flattened output (exports,
  // the preview thumbnail), never previewed live on this canvas.
  layers: LayerDef[];
  onChange: (layers: LayerProps[]) => void;
}

// Converts dotting's own sparse per-layer change payload (a
// Map<rowIndex, Map<columnIndex, {color}>>) into our dense
// [height][width] PixelModifyItem grid. Cells absent from the map are
// "no pixel" (color: ""), matching the rest of the app's convention.
function denseGridFromDottingData(
  data: CanvasDataChangeParams["data"],
  width: number,
  height: number,
): PixelModifyItem[][] {
  const grid: PixelModifyItem[][] = [];
  for (let r = 0; r < height; r++) {
    const row: PixelModifyItem[] = [];
    const rowMap = data.get(r);
    for (let c = 0; c < width; c++) {
      row.push({
        rowIndex: r,
        columnIndex: c,
        color: rowMap?.get(c)?.color ?? "",
      });
    }
    grid.push(row);
  }
  return grid;
}

// dotting has no concept of frames/animation of its own — switching the
// active frame is done imperatively via `setLayers` on the ref (exposed
// here as `loadLayers`), never by remounting <Dotting/> or relying on
// `initLayers` updating reactively (it is only read on mount).
const DottingCanvas = forwardRef<DottingCanvasHandle, DottingCanvasProps>(
  function DottingCanvas(
    { width = 512, height = 512, initLayers, brushTool, brushColor, activeLayerId, layers, onChange },
    outerRef,
  ) {
    const dottingRef = useRef<DottingRef>(null);

    // Keep the latest onChange available to the debounced handler below
    // without making it a dependency of the subscription effect. If the
    // effect depended on `onChange` directly, any caller that doesn't
    // memoize its onChange (a new function identity every render — easy
    // to do by accident) would cause the effect to tear down and rebuild
    // the listener on every unrelated re-render, silently cancelling any
    // in-flight debounce timeout before it fires. That was a real bug:
    // drawing appeared to save sometimes and not others depending on
    // whether anything else re-rendered the parent within 400ms.
    const onChangeRef = useRef(onChange);
    onChangeRef.current = onChange;

    // Latest activeLayerId/layers (metadata) props, for loadLayers() below
    // to read without becoming stale — see onChangeRef above for why a
    // plain-assignment ref, not a dependency, is the right tool here.
    const activeLayerIdRef = useRef(activeLayerId);
    activeLayerIdRef.current = activeLayerId;
    const layersPropRef = useRef(layers);
    layersPropRef.current = layers;

    // The current data for every layer, keyed by layer id. Seeded from
    // initLayers (the frame this canvas was mounted for — remember,
    // switching sprites remounts this component via key={sprite.id}, so
    // this is always the right starting point) and updated incrementally
    // as dataChange events arrive for individual layers. This exists
    // because DottingRef.getLayersAsArray()/getLayers() — the "just ask
    // for the current full state" API — turned out to unconditionally
    // return undefined in practice: both are thin wrappers around an
    // internal `editor` React state value that (in this installed
    // version, 2.1.18) never becomes non-null by the time our listener
    // fires, confirmed by instrumenting it directly. The dataChange event
    // itself, however, carries the changed layer's actual data in its own
    // payload — so we reconstruct full-sprite state from that instead of
    // ever calling back into the ref for it.
    const layersRef = useRef(new Map<string, PixelModifyItem[][]>());
    // The current *set* of layer ids, separate from layersRef's per-layer
    // pixel data. currentLayersArray() below must iterate this — not the
    // initLayers prop directly — because initLayers is a value captured
    // once by the mount-only effect further down; if a layer is added or
    // removed after mount via loadLayers() (#0028), iterating the original
    // initLayers would keep producing the *old* layer set forever, silently
    // dropping the new/removed layer's data from every subsequent save.
    const layerIdsRef = useRef(initLayers.map((l) => l.id));
    const gridWidth = initLayers[0]?.data[0]?.length ?? width;
    const gridHeight = initLayers[0]?.data.length ?? height;

    useEffect(() => {
      layerIdsRef.current = initLayers.map((l) => l.id);
      layersRef.current = new Map(
        initLayers.map((l) => [l.id, l.data]),
      );
      // Only re-seed when the mounted frame's own initLayers identity
      // changes (i.e. never, in practice — a new frame means a remount
      // via key={sprite.id} at the sprite level, or a fresh loadLayers()
      // call within the same sprite, which intentionally does NOT go
      // through this effect — see loadLayers below).
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    function currentLayersArray(): LayerProps[] {
      return layerIdsRef.current.map((id) => ({
        id,
        data: layersRef.current.get(id) ?? [],
      }));
    }

    // Pushes visibility (from sprite metadata, not dotting's own state) into
    // dotting for every layer it currently knows about. Shared by the
    // reactive effect below and by loadLayers()'s post-setLayers fixup —
    // see the comment there for why the latter needs it too.
    function applyLayerVisibility(layerDefs: LayerDef[]) {
      for (const l of layerDefs) {
        if (!layerIdsRef.current.includes(l.id)) continue;
        if (l.visible) {
          dottingRef.current?.showLayer(l.id);
        } else {
          dottingRef.current?.hideLayer(l.id);
        }
      }
    }

    useImperativeHandle(
      outerRef,
      () => ({
        loadLayers(layers: LayerProps[]) {
          layerIdsRef.current = layers.map((l) => l.id);
          layersRef.current = new Map(layers.map((l) => [l.id, l.data]));
          dottingRef.current?.setLayers(layers);
          // dotting's setLayers() constructs brand-new internal layer
          // objects from the given {id, data} pairs alone — no visibility
          // flag exists on LayerProps to preserve, so every layer silently
          // resets to visible, and separately resets dotting's own
          // "current layer" to whichever id is first in the array,
          // regardless of what's actually selected. Both are real, live
          // bugs otherwise: switch frames with a layer hidden, and it
          // reappears; switch frames with a non-topmost layer active, and
          // new strokes silently start landing on the topmost layer
          // instead. Our own effects for both don't re-fire here because
          // neither activeLayerId nor the layers metadata prop necessarily
          // changed — a frame switch changes pixel *data*, not layer
          // *settings* — so this reset has to be undone right where it
          // happens instead.
          applyLayerVisibility(layersPropRef.current);
          if (layerIdsRef.current.includes(activeLayerIdRef.current)) {
            dottingRef.current?.setCurrentLayer(activeLayerIdRef.current);
          }
        },
        getLayers(): LayerProps[] {
          return currentLayersArray();
        },
      }),
      [],
    );

    // Push the active-layer selection into dotting imperatively — it has
    // no UI of its own to change this (LayerPanel is ours), and setLayers/
    // loadLayers above don't touch it, so this needs its own effect.
    //
    // Guarded: dotting's setCurrentLayer throws synchronously ("Layer not
    // found") for any id its own internal dataLayer doesn't have yet. The
    // caller (App's handleAddLayer/handleDeleteLayer) is responsible for
    // sequencing loadLayers() before switching to a new layer id, but this
    // still fires as an effect on every activeLayerId change from whatever
    // App renders — a defensive check against layerIdsRef (the layer set
    // this canvas actually knows about right now) turns any future
    // ordering slip into a silent no-op instead of an uncaught exception
    // that aborts the rest of React's effect flush.
    useEffect(() => {
      if (!layerIdsRef.current.includes(activeLayerId)) return;
      dottingRef.current?.setCurrentLayer(activeLayerId);
    }, [activeLayerId]);

    // Sync each layer's visibility into dotting via showLayer/hideLayer —
    // the LayerPanel checkbox previously only ever patched sprite metadata
    // server-side (used correctly by the export/thumbnail compositor) and
    // never told the live canvas anything, so toggling it visibly did
    // nothing. dotting has no reactive prop for this (LayerProps carries
    // only {id, data}), hence another imperative-ref effect. Guarded the
    // same way as setCurrentLayer above: dotting's showLayer/hideLayer
    // throw synchronously ("Layer not found") for an id its own dataLayer
    // doesn't have yet, which a layer add/delete in flight could trigger.
    useEffect(() => {
      applyLayerVisibility(layers);
    }, [layers]);

    useEffect(() => {
      const ref = dottingRef.current;
      if (!ref) return;

      let timeout: ReturnType<typeof setTimeout> | undefined;
      const handleChange = (params: CanvasDataChangeParams) => {
        layersRef.current.set(
          params.layerId,
          denseGridFromDottingData(params.data, gridWidth, gridHeight),
        );
        // dotting also fires this event for its own internal, non-user
        // data sync (e.g. once on mount while it initializes from
        // initLayers) with isLocalChange: false — and its payload for
        // that particular event is empty regardless of what initLayers
        // actually contained. Saving unconditionally on every event was a
        // real, destructive bug: simply reopening a sprite with existing
        // drawn pixels would silently overwrite them with a blank frame,
        // because that spurious mount-time event would schedule (and
        // eventually fire) a save of empty data over real content. Only
        // schedule a save for changes the user actually made.
        if (!params.isLocalChange) return;
        if (timeout) clearTimeout(timeout);
        timeout = setTimeout(() => {
          onChangeRef.current(currentLayersArray());
        }, 400);
      };

      ref.addDataChangeListener(handleChange);
      return () => {
        if (timeout) clearTimeout(timeout);
        try {
          ref.removeDataChangeListener(handleChange);
        } catch {
          // dotting can tear down its internal editor before this cleanup
          // runs (observed during React StrictMode's dev-only double
          // mount/unmount cycle) — nothing left to detach from in that case.
        }
      };
      // Mount/unmount only — see onChangeRef above for why onChange itself
      // must not be a dependency here.
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
      // dotting has two separate opaque fills that both need disabling to
      // get a truly transparent canvas:
      //  - `backgroundColor` is a plain CSS background-color on a canvas
      //    *element* (the area outside the declared grid, default #999)
      //    — "transparent" works fine there, it's an ordinary CSS value.
      //  - `defaultPixelColor` is used inside dotting's own 2D-canvas
      //    render code as `if (this.defaultPixelColor) { ctx.fillStyle =
      //    this.defaultPixelColor; ctx.fillRect(...) }` to paint the
      //    entire grid *interior* (default #fff) before drawing actual
      //    pixels on top. Passing "transparent" there still runs
      //    fillRect with that as the canvas fillStyle, which is a less
      //    certain no-op than skipping the fill outright — so pass ""
      //    instead: falsy, so dotting's own truthiness check skips the
      //    fillRect call entirely, guaranteed to leave the grid's empty
      //    cells untouched (and thus transparent) rather than depending
      //    on how a given browser's canvas resolves the literal string
      //    "transparent" as a fill color.
      <div className="dotting-canvas-checkerboard">
        <Dotting
          ref={dottingRef}
          width={width}
          height={height}
          initLayers={initLayers}
          brushTool={brushTool}
          brushColor={brushColor}
          backgroundColor="transparent"
          defaultPixelColor=""
          isGridVisible
          isPanZoomable
          isGridFixed
          initAutoScale
          gridSquareLength={20}
        />
      </div>
    );
  },
);

export default DottingCanvas;
export { BrushTool };
export type { LayerProps };
