// dotting@2.1.18's package.json "types" field points at
// build/src/index.d.ts, which doesn't exist in the published package —
// the real declarations live at build/index.d.ts. Redirect the type-only
// resolution here (an ambient module declaration only affects
// type-checking; the actual runtime import of "dotting" still resolves
// normally via package.json main/module, so this doesn't touch bundling).
declare module "dotting" {
  export * from "dotting/build/index";
}
