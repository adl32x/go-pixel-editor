import { useSyncExternalStore } from "react";
import type { LoopMode, Sprite, SpriteSummary } from "../types";

export interface PlaybackState {
  playing: boolean;
  entryIndex: number;
  fps: number;
  loop: LoopMode;
}

export interface StoreState {
  sprites: SpriteSummary[];
  selectedSpriteId: string | null;
  sprite: Sprite | null;
  selectedFrameId: string | null;
  selectedClipName: string | null;
  playback: PlaybackState;
}

const initialState: StoreState = {
  sprites: [],
  selectedSpriteId: null,
  sprite: null,
  selectedFrameId: null,
  selectedClipName: null,
  playback: { playing: false, entryIndex: 0, fps: 4, loop: "forward" },
};

let state: StoreState = initialState;
const listeners = new Set<() => void>();

function emit() {
  for (const listener of listeners) listener();
}

export const spriteStore = {
  getState(): StoreState {
    return state;
  },
  subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
  set(partial: Partial<StoreState>) {
    state = { ...state, ...partial };
    emit();
  },
  setPlayback(partial: Partial<PlaybackState>) {
    state = { ...state, playback: { ...state.playback, ...partial } };
    emit();
  },
};

export function useSpriteStore(): StoreState {
  return useSyncExternalStore(spriteStore.subscribe, spriteStore.getState);
}
