import { readFileSync, readdirSync } from "node:fs";
import { join, relative, resolve, sep } from "node:path";
import { optimize } from "svgo";
import type { CustomPlugin } from "svgo";
import type { Plugin, ViteDevServer } from "vite";

// Preserve the existing virtual imports and icon-* IDs without svg-baker's
// unsupported PostCSS 5 and micromatch 3 dependency trees.
export function symbolFromSVG(source: string, id: string): string {
  const toSymbol: CustomPlugin = {
    name: "local-icon-symbol",
    fn: () => ({
      element: {
        enter(node, parent) {
          if (parent.type !== "root" || node.name !== "svg") return;
          node.name = "symbol";
          node.attributes.id = id;
          if (!node.attributes.viewBox && node.attributes.width && node.attributes.height) {
            const width = Number.parseFloat(node.attributes.width);
            const height = Number.parseFloat(node.attributes.height);
            if (width > 0 && height > 0) node.attributes.viewBox = `0 0 ${width} ${height}`;
          }
          delete node.attributes.xmlns;
          delete node.attributes.width;
          delete node.attributes.height;
        },
      },
    }),
  };
  return optimize(source, {
    plugins: [
      "removeDoctype", "removeXMLProcInst", "removeComments", "removeScripts",
      { name: "prefixIds", params: { prefix: id, delim: "__" } },
      toSymbol,
    ],
  }).data;
}

export function createSvgIconsPlugin(directory: string): Plugin {
  const root = resolve(directory);
  const register = "virtual:svg-icons-register";
  const names = "virtual:svg-icons-names";
  function iconFiles(dir: string): string[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
      const path = join(dir, entry.name);
      return entry.isDirectory() ? iconFiles(path) : entry.isFile() && path.endsWith(".svg") ? [path] : [];
    }).sort();
  }
  function icons() {
    return iconFiles(root).map(path => ({
      path,
      id: "icon-" + relative(root, path).slice(0, -4).split(sep).join("-"),
    }));
  }
  function reload(server: ViteDevServer, path: string) {
    const local = relative(root, path);
    if (local.startsWith("..") || !local.endsWith(".svg")) return;
    for (const id of [register, names]) {
      const module = server.moduleGraph.getModuleById("\0" + id);
      if (module) server.moduleGraph.invalidateModule(module);
    }
    server.ws.send({ type: "full-reload" });
  }
  return {
    name: "local-svg-icons",
    resolveId(id) { if (id === register || id === names) return "\0" + id; },
    load(id) {
      if (id !== "\0" + register && id !== "\0" + names) return;
      const files = icons();
      for (const file of files) this.addWatchFile(file.path);
      if (id === "\0" + names) return "export default " + JSON.stringify(files.map(file => file.id));
      const markup = files.map(file => symbolFromSVG(readFileSync(file.path, "utf8"), file.id)).join("");
      return `function mount(){
        if(document.getElementById('__svg__icons__dom__')) return;
        const sprite=document.createElementNS('http://www.w3.org/2000/svg','svg');
        sprite.id='__svg__icons__dom__';sprite.setAttribute('aria-hidden','true');
        sprite.setAttribute('width','0');sprite.setAttribute('height','0');
        sprite.style.cssText='position:absolute;overflow:hidden';
        sprite.innerHTML=${JSON.stringify(markup)};
        document.body.prepend(sprite);
      }
      if(document.readyState==='loading') document.addEventListener('DOMContentLoaded',mount,{once:true});else mount();`;
    },
    configureServer(server) {
      server.watcher.add(root);
      const changed = (path: string) => reload(server, path);
      for (const event of ["add", "change", "unlink"] as const) server.watcher.on(event, changed);
      server.httpServer?.once("close", () => {
        for (const event of ["add", "change", "unlink"] as const) server.watcher.off(event, changed);
      });
    },
  };
}
