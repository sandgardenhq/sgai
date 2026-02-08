import { resolve, join } from "path";
import { watch, mkdirSync, existsSync } from "fs";
import postcss from "postcss";
import tailwindcss from "@tailwindcss/postcss";

const API_TARGET = process.env.API_TARGET ?? "http://127.0.0.1:8181";
const DEV_PORT = parseInt(process.env.DEV_PORT ?? "5173", 10);

const cssInputPath = resolve("./src/index.css");
const devCssOutputPath = resolve("./src/.dev-compiled.css");

async function compileTailwindCSS(): Promise<void> {
  const source = await Bun.file(cssInputPath).text();
  const result = await postcss([tailwindcss()]).process(source, {
    from: cssInputPath,
  });
  await Bun.write(devCssOutputPath, result.css);
}

await compileTailwindCSS();

const devIndexHtmlPath = resolve("./.dev-index.html");
const originalHtml = await Bun.file("./index.html").text();
const devHtml = originalHtml.replace(
  '<link rel="stylesheet" href="/src/index.css" />',
  '<link rel="stylesheet" href="/src/.dev-compiled.css" />',
);
await Bun.write(devIndexHtmlPath, devHtml);

const devIndex = await import("./.dev-index.html");

const srcDir = resolve("./src");
watch(srcDir, { recursive: true }, async (_event, filename) => {
  if (filename && !filename.startsWith(".dev-") &&
      (filename.endsWith(".css") || filename.endsWith(".tsx") || filename.endsWith(".ts"))) {
    try {
      await compileTailwindCSS();
    } catch (err) {
      console.error("CSS compilation error:", err);
    }
  }
});

async function proxyToAPI(
  request: Request,
  pathname: string,
): Promise<Response> {
  const url = new URL(
    pathname + new URL(request.url).search,
    API_TARGET,
  );
  try {
    const proxyResponse = await fetch(url.toString(), {
      method: request.method,
      headers: request.headers,
      body:
        request.method !== "GET" && request.method !== "HEAD"
          ? await request.blob()
          : undefined,
    });
    return new Response(proxyResponse.body, {
      status: proxyResponse.status,
      statusText: proxyResponse.statusText,
      headers: proxyResponse.headers,
    });
  } catch {
    return new Response("API server unavailable", { status: 502 });
  }
}

const server = Bun.serve({
  port: DEV_PORT,
  development: {
    hmr: true,
    console: true,
  },
  routes: {
    "/*": devIndex.default,
  },
  async fetch(request) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    if (pathname.startsWith("/api/")) {
      return proxyToAPI(request, pathname);
    }

    return new Response(null, { status: 404 });
  },
});

console.log(`Dev server running at http://localhost:${server.port}`);
console.log(`Proxying /api/* to ${API_TARGET}`);
console.log(`Tailwind CSS compiled and watching for changes`);
