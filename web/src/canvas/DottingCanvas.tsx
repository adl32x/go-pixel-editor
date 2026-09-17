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
          onChange(ref.getLayersAsArray());
        }, 400);
      };

      ref.addDataChangeListener(handleChange);
      return () => {
        if (timeout) clearTimeout(timeout);
        ref.removeDataChangeListener(handleChange);
      };
    }, [onChange]);

    return (
      <Dotting
        ref={dottingRef}
        width={width}
        height={height}
        initLayers={initLayers}
        brushTool={brushTool}
        brushColor={brushColor}
        isGridVisible
        isPanZoomable
        gridSquareLength={20}
      />
    );
  },
);

export default DottingCanvas;
export { BrushTool };
export type { LayerProps };
