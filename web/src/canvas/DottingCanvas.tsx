import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { BrushTool, Dotting, type DottingRef, type LayerProps } from "dotting";

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
  onChange: (layers: LayerProps[]) => void;
}

// dotting has no concept of frames/animation of its own — switching the
// active frame is done imperatively via `setLayers` on the ref (exposed
// here as `loadLayers`), never by remounting <Dotting/> or relying on
// `initLayers` updating reactively (it is only read on mount).
const DottingCanvas = forwardRef<DottingCanvasHandle, DottingCanvasProps>(
  function DottingCanvas(
    { width = 512, height = 512, initLayers, brushTool, brushColor, onChange },
    outerRef,
  ) {
    const dottingRef = useRef<DottingRef>(null);

    useImperativeHandle(
      outerRef,
      () => ({
        loadLayers(layers: LayerProps[]) {
          dottingRef.current?.setLayers(layers);
        },
        getLayers(): LayerProps[] {
          return dottingRef.current?.getLayersAsArray() ?? [];
        },
      }),
      [],
    );

    useEffect(() => {
      const ref = dottingRef.current;
      if (!ref) return;

      let timeout: ReturnType<typeof setTimeout> | undefined;
      const handleChange = () => {
        if (timeout) clearTimeout(timeout);
        timeout = setTimeout(() => {
          // getLayersAsArray() can return undefined if dotting's internal
          // editor has already torn down by the time this debounced
          // callback fires (e.g. a frame/sprite switch mid-debounce) —
          // nothing to persist in that case.
          const layers = ref.getLayersAsArray();
          if (layers) onChange(layers);
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
    }, [onChange]);

    return (
      // dotting has two separate opaque fills that both need disabling to
      // get a truly transparent canvas: `backgroundColor` (the area
      // outside the declared grid, default #999) and `defaultPixelColor`
      // (every cell *inside* the grid with no color, default #fff,
      // painted underneath every layer on every render regardless of
      // backgroundColor). Setting both to "transparent" makes each a
      // no-op fill (canvas resolves "transparent" to alpha 0) so the
      // checkerboard behind the canvas (see .dotting-canvas-checkerboard)
      // shows through instead — matching sprites' actual default (a new
      // frame's cells all start unset, i.e. transparent, not any color).
      <div className="dotting-canvas-checkerboard">
        <Dotting
          ref={dottingRef}
          width={width}
          height={height}
          initLayers={initLayers}
          brushTool={brushTool}
          brushColor={brushColor}
          backgroundColor="transparent"
          defaultPixelColor="transparent"
          isGridVisible
          isPanZoomable
          isGridFixed
          gridSquareLength={20}
        />
      </div>
    );
  },
);

export default DottingCanvas;
export { BrushTool };
export type { LayerProps };
