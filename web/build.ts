// Bun does the bundling: HTML-import bundler resolves the <script>/<link>
// references inside index.html, fingerprints assets, and rewrites the HTML.
// Output feeds internal/server/dist, embedded via go:embed in the Go binary.
export {};

const result = await Bun.build({
  entrypoints: ["./index.html"],
  outdir: "../internal/server/dist",
  minify: true,
  sourcemap: "linked",
});

if (!result.success) {
  console.error(`Build failed with ${result.logs.length} error(s):`);
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

for (const output of result.outputs) {
  console.log(`built ${output.path} (${output.kind})`);
}
