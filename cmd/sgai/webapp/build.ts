import { readFileSync, writeFileSync } from "fs";
import { join, relative, resolve } from "path";
import postcss from "postcss";
import tailwindcss from "@tailwindcss/postcss";

const startTime = performance.now();
const distDir = resolve("./dist");

const tailwindPlugin: import("bun").BunPlugin = {
  name: "tailwind-css",
  setup(build) {
    build.onLoad({ filter: /\.css$/ }, async (args) => {
      const source = await Bun.file(args.path).text();
      const result = await postcss([tailwindcss()]).process(source, {
        from: args.path,
      });
      return {
        contents: result.css,
        loader: "css",
      };
    });
  },
};

const result = await Bun.build({
  entrypoints: ["./src/main.tsx"],
  outdir: "./dist",
  splitting: true,
  minify: true,
  sourcemap: "linked",
  target: "browser",
  plugins: [tailwindPlugin],
  naming: {
    entry: "assets/[name]-[hash].[ext]",
    chunk: "assets/[name]-[hash].[ext]",
    asset: "assets/[name]-[hash].[ext]",
  },
});

if (!result.success) {
  console.error("Build failed:");
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

function relPath(absPath: string): string {
  return "/" + relative(distDir, absPath);
}

const cssOutput = result.outputs.find((o) => o.path.endsWith(".css"));
const jsEntry = result.outputs.find(
  (o) => o.kind === "entry-point" && o.path.endsWith(".js"),
);

const indexHtml = readFileSync("./index.html", "utf-8");

const cssTag = cssOutput
  ? `<link rel="stylesheet" href="${relPath(cssOutput.path)}" />`
  : "";

const jsTag = jsEntry
  ? `<script type="module" src="${relPath(jsEntry.path)}"></script>`
  : "";

const outputHtml = indexHtml
  .replace('<link rel="stylesheet" href="/src/index.css" />', cssTag)
  .replace(
    '<script type="module" src="/src/main.tsx"></script>',
    jsTag,
  );

writeFileSync("./dist/index.html", outputHtml);

const manifest = {
  buildTime: new Date().toISOString(),
  outputs: result.outputs.map((o) => ({
    path: o.path,
    kind: o.kind,
    size: o.size,
  })),
};
writeFileSync("./dist/manifest.json", JSON.stringify(manifest, null, 2));

const elapsed = (performance.now() - startTime).toFixed(0);
console.log(`Build complete in ${elapsed}ms`);
console.log(`  Output files: ${result.outputs.length}`);
for (const output of result.outputs) {
  const sizeKB = (output.size / 1024).toFixed(1);
  console.log(`  ${output.path} (${sizeKB} KB)`);
}
