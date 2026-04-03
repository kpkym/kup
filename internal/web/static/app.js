const { createApp, ref, computed, watch, nextTick, onMounted } = Vue;

const EXT_TYPES = {
  video: ["mp4", "webm", "ogg", "mov", "mkv"],
  image: ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "ico"],
  audio: ["mp3", "wav", "flac", "aac", "m4a"],
  text: [
    "txt", "md", "json", "xml", "yaml", "yml", "toml", "ini", "cfg",
    "log", "csv", "js", "ts", "go", "py", "sh", "bash", "html", "css",
    "sql", "rs", "c", "h", "cpp", "java", "rb", "php",
  ],
};

const ICON_MAP = {
  video: ["mp4", "webm", "ogg", "mov", "mkv", "avi"],
  image: ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "ico"],
  audio: ["mp3", "wav", "flac", "aac", "m4a"],
  code: ["js", "ts", "go", "py", "rs", "c", "h", "cpp", "java", "rb", "php", "sh", "bash"],
  doc: ["md", "txt", "pdf", "doc", "docx"],
  data: ["json", "xml", "yaml", "yml", "toml", "csv", "sql"],
  archive: ["zip", "tar", "gz", "bz2", "xz", "7z", "rar"],
  config: ["ini", "cfg", "conf", "env"],
};

const ICON_CHARS = {
  dir: "\u{1F4C1}", video: "\u{1F3AC}", image: "\u{1F5BC}", audio: "\u{1F3B5}",
  code: "\u{1F4DD}", doc: "\u{1F4C4}", data: "\u{1F4CA}", archive: "\u{1F4E6}",
  config: "\u2699", file: "\u{1F4C4}",
};

function getExt(name) {
  return name.includes(".") ? name.split(".").pop().toLowerCase() : "";
}

function fileIcon(name, type) {
  if (type === "dir") return ICON_CHARS.dir;
  const ext = getExt(name);
  for (const [cat, exts] of Object.entries(ICON_MAP)) {
    if (exts.includes(ext)) return ICON_CHARS[cat];
  }
  return ICON_CHARS.file;
}

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function formatDate(iso) {
  const d = new Date(iso);
  const diff = Date.now() - d;
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}d ago`;
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function formatDateFull(iso) {
  return new Date(iso).toLocaleString();
}

async function apiFetch(path) {
  const res = await fetch(path);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

// Browser-persistent cache using localStorage
const CACHE_PREFIX = "kup:";
const CACHE_SNAP_KEY = CACHE_PREFIX + "snapshots";

function cacheGet(key) {
  try {
    const raw = localStorage.getItem(CACHE_PREFIX + key);
    return raw ? JSON.parse(raw) : null;
  } catch { return null; }
}

function cacheSet(key, value) {
  try {
    localStorage.setItem(CACHE_PREFIX + key, JSON.stringify(value));
  } catch { /* storage full — ignore */ }
}

function cacheClearAll() {
  const keys = [];
  for (let i = 0; i < localStorage.length; i++) {
    const k = localStorage.key(i);
    if (k && k.startsWith(CACHE_PREFIX)) keys.push(k);
  }
  keys.forEach((k) => localStorage.removeItem(k));
}

function cacheCount() {
  let n = 0;
  for (let i = 0; i < localStorage.length; i++) {
    if (localStorage.key(i)?.startsWith(CACHE_PREFIX)) n++;
  }
  return n;
}

async function fetchSnapshots() {
  const cached = cacheGet("snapshots");
  if (cached) return cached;
  const data = await apiFetch("/api/snapshots");
  cacheSet("snapshots", data);
  return data;
}

async function fetchLs(id, dirPath) {
  const key = `ls:${id}:${dirPath}`;
  const cached = cacheGet(key);
  if (cached) return cached;
  const entries = await apiFetch(`/api/snapshots/${id}/ls?path=${encodeURIComponent(dirPath)}`);
  cacheSet(key, entries);
  return entries;
}

function sortEntries(entries) {
  return [...entries].sort((a, b) => {
    if (a.type === "dir" && b.type !== "dir") return -1;
    if (a.type !== "dir" && b.type === "dir") return 1;
    return a.name.localeCompare(b.name);
  });
}

function pathToSegments(dirPath) {
  const parts = dirPath.split("/").filter(Boolean);
  const result = ["/"];
  let cum = "/";
  for (const p of parts) {
    cum += p + "/";
    result.push(cum);
  }
  return result;
}

createApp({
  setup() {
    const view = ref("snapshots");
    const snapId = ref("");

    // Snapshots
    const snapshots = ref([]);
    const snapshotsLoading = ref(false);
    const snapshotsError = ref("");

    // Columns
    const columns = ref([]);     // [{ path, loading, error, entries, sorted, selected }]
    const previewFile = ref(null);     // the file being loaded (header shows this)
    const readyPreview = ref(null);   // the file whose content is ready (body shows this)
    const previewLoading = ref(false);
    const textContent = ref("");
    const textLoading = ref(false);
    const columnsEl = ref(null);

    const previewType = computed(() => {
      if (!readyPreview.value) return null;
      const ext = getExt(readyPreview.value.name);
      for (const [type, exts] of Object.entries(EXT_TYPES)) {
        if (exts.includes(ext)) return type;
      }
      return null;
    });

    function dumpUrl(file) {
      return `/api/snapshots/${snapId.value}/dump?path=${encodeURIComponent(file.path)}`;
    }

    function entryHref(entry, colPath) {
      if (entry.type === "dir") {
        return `#/snap/${snapId.value}/browse?path=${encodeURIComponent(entry.path + "/")}`;
      }
      return `#/snap/${snapId.value}/browse?path=${encodeURIComponent(colPath)}&selected=${encodeURIComponent(entry.name)}`;
    }

    // --- Load snapshots ---

    async function loadSnapshots() {
      snapshotsLoading.value = true;
      snapshotsError.value = "";
      try {
        const data = await fetchSnapshots();
        snapshots.value = (data || []).sort((a, b) => new Date(b.time) - new Date(a.time));
        cachedEntries.value = cacheCount();
      } catch (e) {
        snapshotsError.value = e.message;
      } finally {
        snapshotsLoading.value = false;
      }
    }

    // --- Columns logic ---

    async function loadColumn(dirPath) {
      const col = { path: dirPath, loading: true, error: null, entries: [], sorted: [], selected: null };
      try {
        const entries = await fetchLs(snapId.value, dirPath);
        col.entries = entries;
        col.sorted = sortEntries(entries);
      } catch (e) {
        col.error = e.message;
      }
      col.loading = false;
      cachedEntries.value = cacheCount();
      return col;
    }

    async function navigateBrowser(id, dirPath, selectedFile) {
      snapId.value = id;
      const colPaths = pathToSegments(dirPath);

      // Figure out which columns we can reuse
      let reuseCount = 0;
      for (let i = 0; i < Math.min(columns.value.length, colPaths.length); i++) {
        if (columns.value[i].path === colPaths[i]) {
          reuseCount = i + 1;
        } else {
          break;
        }
      }

      // Trim extra columns
      columns.value.length = reuseCount;

      // Pre-compute selections for all columns
      const selectedInCol = {};
      for (let i = 0; i < colPaths.length - 1; i++) {
        selectedInCol[colPaths[i]] = colPaths[i + 1].split("/").filter(Boolean).pop();
      }
      if (selectedFile) {
        selectedInCol[colPaths[colPaths.length - 1]] = selectedFile;
      }

      // Update selections on reused columns
      for (let i = 0; i < reuseCount; i++) {
        columns.value[i].selected = selectedInCol[colPaths[i]] || null;
      }

      // Add loading placeholders for new columns with selections pre-set
      const newPaths = colPaths.slice(reuseCount);
      for (const p of newPaths) {
        columns.value.push({ path: p, loading: true, error: null, entries: [], sorted: [], selected: selectedInCol[p] || null });
      }

      // Scroll to loading columns
      await nextTick();
      scrollColumnsRight();

      // Fetch new columns in parallel
      const results = await Promise.all(newPaths.map((p) => loadColumn(p)));

      // Replace placeholders with real data
      for (let i = 0; i < results.length; i++) {
        const idx = reuseCount + i;
        const result = results[i];
        // Preserve selection that was set above
        result.selected = columns.value[idx].selected;
        columns.value[idx] = result;
      }

      // Auto-expand: if the last column has exactly one dir entry, keep drilling
      if (!selectedFile) {
        let lastCol = columns.value[columns.value.length - 1];
        while (
          lastCol &&
          !lastCol.error &&
          lastCol.entries.length === 1 &&
          lastCol.entries[0].type === "dir"
        ) {
          const onlyDir = lastCol.entries[0];
          lastCol.selected = onlyDir.name;
          const nextPath = onlyDir.path + "/";

          // Add loading placeholder
          columns.value.push({ path: nextPath, loading: true, error: null, entries: [], sorted: [], selected: null });
          await nextTick();
          scrollColumnsRight();

          const nextCol = await loadColumn(nextPath);
          columns.value[columns.value.length - 1] = nextCol;
          lastCol = nextCol;
        }

        // Update the hash to reflect the auto-expanded path without triggering re-render
        if (columns.value.length > 0) {
          const deepestPath = columns.value[columns.value.length - 1].path;
          if (deepestPath !== dirPath) {
            const newHash = `#/snap/${id}/browse?path=${encodeURIComponent(deepestPath)}`;
            history.replaceState(null, "", newHash);
          }
        }
      }

      // Handle file preview — only update when a file is explicitly selected
      if (selectedFile) {
        const lastCol = columns.value[columns.value.length - 1];
        const file = lastCol.entries.find((e) => e.name === selectedFile);
        if (file) {
          setPreview(file);
        }
      }

      await nextTick();
      scrollColumnsRight();
    }

    function onEntryClick(entry, colPath, colIndex) {
      if (entry.type === "dir") {
        const newPath = entry.path + "/";
        location.hash = `#/snap/${snapId.value}/browse?path=${encodeURIComponent(newPath)}`;
      } else {
        location.hash = `#/snap/${snapId.value}/browse?path=${encodeURIComponent(colPath)}&selected=${encodeURIComponent(entry.name)}`;
      }
    }

    function setPreview(file) {
      // If same file, do nothing
      if (previewFile.value && previewFile.value.path === file.path) return;

      previewFile.value = file;
      previewLoading.value = true;

      const ext = getExt(file.name);
      if (EXT_TYPES.text.includes(ext)) {
        textLoading.value = true;
        fetch(dumpUrl(file))
          .then((r) => r.text())
          .then((text) => {
            // Only apply if still the current file
            if (previewFile.value?.path !== file.path) return;
            textContent.value = text;
            textLoading.value = false;
            readyPreview.value = file;
            previewLoading.value = false;
          })
          .catch(() => {
            if (previewFile.value?.path !== file.path) return;
            textLoading.value = false;
            readyPreview.value = file;
            previewLoading.value = false;
          });
      } else if (EXT_TYPES.image.includes(ext)) {
        // Preload image, swap when ready
        const img = new Image();
        img.onload = img.onerror = () => {
          if (previewFile.value?.path !== file.path) return;
          readyPreview.value = file;
          previewLoading.value = false;
        };
        img.src = dumpUrl(file);
      } else {
        // Video, audio, no-preview: swap immediately
        readyPreview.value = file;
        previewLoading.value = false;
      }
    }

    function scrollColumnsRight() {
      if (columnsEl.value) {
        columnsEl.value.scrollLeft = columnsEl.value.scrollWidth;
      }
    }

    // --- Router ---

    function onHashChange() {
      const hash = location.hash || "#/";
      const [routePart, queryPart] = hash.slice(1).split("?");
      const params = new URLSearchParams(queryPart || "");

      if (routePart.match(/^\/snap\/[^/]+\/browse/)) {
        view.value = "columns";
        const id = routePart.split("/")[2];
        const dirPath = params.get("path") || "/";
        const selected = params.get("selected") || "";
        navigateBrowser(id, dirPath, selected);
      } else {
        view.value = "snapshots";
        previewFile.value = null;
        readyPreview.value = null;
        columns.value = [];
        if (!snapshots.value.length) loadSnapshots();
      }
    }

    const cachedEntries = ref(cacheCount());

    function clearCache() {
      cacheClearAll();
      cachedEntries.value = 0;
      snapshots.value = [];
      columns.value = [];
      previewFile.value = null;
      readyPreview.value = null;
      location.hash = "#/";
      // If already on #/, hashchange won't fire — reload manually
      loadSnapshots();
    }

    onMounted(() => {
      window.addEventListener("hashchange", onHashChange);
      onHashChange();
    });

    return {
      view, snapId,
      snapshots, snapshotsLoading, snapshotsError,
      columns, columnsEl, previewFile, readyPreview, previewType, previewLoading,
      textContent, textLoading, cachedEntries,
      fileIcon, formatBytes, formatDate, formatDateFull,
      dumpUrl, entryHref, onEntryClick, clearCache,
    };
  },
}).mount("#app");
