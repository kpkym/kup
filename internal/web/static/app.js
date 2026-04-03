const $ = (sel) => document.querySelector(sel);
const app = $("#app");

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function formatDate(iso) {
  return new Date(iso).toLocaleString();
}

async function api(path) {
  const res = await fetch(path);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

// Router
function navigate() {
  const hash = location.hash || "#/";
  const parts = hash.slice(1).split("?");
  const route = parts[0];
  const params = new URLSearchParams(parts[1] || "");

  if (route === "/") {
    renderSnapshots();
  } else if (route.match(/^\/snap\/[^/]+\/browse/)) {
    const id = route.split("/")[2];
    const dirPath = params.get("path") || "/";
    renderBrowser(id, dirPath);
  } else if (route.match(/^\/snap\/[^/]+\/preview/)) {
    const id = route.split("/")[2];
    const filePath = params.get("path") || "";
    const size = params.get("size") || "";
    renderPreview(id, filePath, size);
  } else {
    renderSnapshots();
  }
}

async function renderSnapshots() {
  app.innerHTML = `<h1>Snapshots</h1><p class="loading">Loading...</p>`;
  try {
    const snaps = await api("/api/snapshots");
    if (!snaps || snaps.length === 0) {
      app.innerHTML = `<h1>Snapshots</h1><p>No snapshots found.</p>`;
      return;
    }
    const rows = snaps
      .sort((a, b) => new Date(b.time) - new Date(a.time))
      .map(
        (s) => `
      <tr>
        <td><a href="#/snap/${s.short_id}/browse?path=/">${s.short_id}</a></td>
        <td>${formatDate(s.time)}</td>
        <td>${s.hostname || ""}</td>
        <td>${(s.paths || []).join(", ")}</td>
      </tr>`
      )
      .join("");
    app.innerHTML = `
      <h1>Snapshots</h1>
      <table>
        <thead><tr><th>ID</th><th>Time</th><th>Host</th><th>Paths</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>`;
  } catch (e) {
    app.innerHTML = `<h1>Snapshots</h1><p>Error: ${e.message}</p>`;
  }
}

function buildBreadcrumb(id, dirPath) {
  const parts = dirPath.split("/").filter(Boolean);
  let crumbs = `<a href="#/snap/${id}/browse?path=/">/</a>`;
  let cumulative = "/";
  for (const part of parts) {
    cumulative += part + "/";
    crumbs += `<span>/</span><a href="#/snap/${id}/browse?path=${encodeURIComponent(cumulative)}">${part}</a>`;
  }
  return `<div class="breadcrumb">${crumbs}</div>`;
}

async function renderBrowser(id, dirPath) {
  app.innerHTML = `${buildBreadcrumb(id, dirPath)}<p class="loading">Loading...</p>`;
  try {
    const entries = await api(
      `/api/snapshots/${id}/ls?path=${encodeURIComponent(dirPath)}`
    );
    const rows = entries
      .sort((a, b) => {
        if (a.type === "dir" && b.type !== "dir") return -1;
        if (a.type !== "dir" && b.type === "dir") return 1;
        return a.name.localeCompare(b.name);
      })
      .map((e) => {
        const icon = e.type === "dir" ? "📁" : "📄";
        const href =
          e.type === "dir"
            ? `#/snap/${id}/browse?path=${encodeURIComponent(e.path + "/")}`
            : `#/snap/${id}/preview?path=${encodeURIComponent(e.path)}&size=${e.size || 0}`;
        const size = e.type === "dir" ? "" : formatBytes(e.size || 0);
        return `<tr><td><a href="${href}">${icon} ${e.name}</a></td><td class="size">${size}</td><td class="size">${e.mtime ? formatDate(e.mtime) : ""}</td></tr>`;
      })
      .join("");
    app.innerHTML = `
      ${buildBreadcrumb(id, dirPath)}
      <table>
        <thead><tr><th>Name</th><th>Size</th><th>Modified</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>`;
  } catch (e) {
    app.innerHTML = `${buildBreadcrumb(id, dirPath)}<p>Error: ${e.message}</p>`;
  }
}

function renderPreview(id, filePath, size) {
  const name = filePath.split("/").pop();
  const ext = name.includes(".") ? name.split(".").pop().toLowerCase() : "";
  const dumpUrl = `/api/snapshots/${id}/dump?path=${encodeURIComponent(filePath)}`;

  const dirPath = filePath.substring(0, filePath.lastIndexOf("/") + 1) || "/";
  const backLink = `<a class="back" href="#/snap/${id}/browse?path=${encodeURIComponent(dirPath)}">← Back</a>`;

  const videoExts = ["mp4", "webm", "ogg", "mov", "mkv"];
  const imageExts = ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "ico"];
  const audioExts = ["mp3", "wav", "ogg", "flac", "aac", "m4a"];
  const textExts = [
    "txt", "md", "json", "xml", "yaml", "yml", "toml", "ini", "cfg",
    "log", "csv", "js", "ts", "go", "py", "sh", "bash", "html", "css",
    "sql", "rs", "c", "h", "cpp", "java", "rb", "php",
  ];

  let preview;
  if (videoExts.includes(ext)) {
    preview = `<video controls autoplay><source src="${dumpUrl}">Your browser does not support video.</video>`;
  } else if (imageExts.includes(ext)) {
    preview = `<img src="${dumpUrl}" alt="${name}">`;
  } else if (audioExts.includes(ext)) {
    preview = `<audio controls autoplay><source src="${dumpUrl}"></audio>`;
  } else if (textExts.includes(ext)) {
    preview = `<pre class="loading">Loading...</pre>`;
    fetch(dumpUrl)
      .then((r) => r.text())
      .then((text) => {
        const pre = app.querySelector("pre");
        if (pre) {
          pre.classList.remove("loading");
          pre.textContent = text;
        }
      });
  } else {
    preview = `<p><a href="${dumpUrl}" download="${name}">Download ${name}</a> (${formatBytes(Number(size))})</p>`;
  }

  app.innerHTML = `
    ${backLink}
    <h1>${name}</h1>
    <div class="preview">${preview}</div>`;
}

window.addEventListener("hashchange", navigate);
navigate();
