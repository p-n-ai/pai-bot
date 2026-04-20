(function () {
  "use strict";

  var existing = window.PAIWidget;
  if (existing && typeof existing.destroy === "function") {
    return;
  }

  var script = document.currentScript || document.querySelector('script[src*="embed.js"]');
  var origin = new URL(script && script.src ? script.src : window.location.href).origin;
  var dataset = script ? script.dataset || {} : {};
  var root = document.createElement("div");
  var shadow = root.attachShadow ? root.attachShadow({ mode: "open" }) : root;
  var state = { open: false, ready: false };

  root.id = "pai-widget-root";
  document.body.appendChild(root);

  function clean(value, fallback) {
    return String(value || fallback).replace(/[\u0000-\u001f<>]/g, "").trim();
  }

  function isLeft() {
    return dataset.position === "left";
  }

  function widgetURL() {
    var url = new URL("/widget", origin);
    url.searchParams.set("tenant", clean(dataset.tenant, "public"));
    url.searchParams.set("title", clean(dataset.title, "P&AI Tutor"));
    url.searchParams.set("theme", dataset.theme === "dark" ? "dark" : "light");
    if (dataset.apiOrigin) {
      url.searchParams.set("apiOrigin", dataset.apiOrigin);
    }
    return url.toString();
  }

  function emit(name) {
    window.dispatchEvent(
      new CustomEvent("pai-widget:" + name, {
        detail: {
          tenant: clean(dataset.tenant, "public"),
          open: state.open,
          ready: state.ready,
        },
      }),
    );
  }

  var style = document.createElement("style");
  style.textContent =
    ":host{all:initial}" +
    ".pai-launcher{position:fixed;z-index:2147483647;right:24px;bottom:24px;width:56px;height:56px;border:0;border-radius:50%;background:#1f2937;color:#fff;box-shadow:0 16px 40px rgba(15,23,42,.24);cursor:pointer;display:grid;place-items:center;font-family:Inter,system-ui,sans-serif}" +
    ".pai-launcher:hover{background:#111827}" +
    ".pai-launcher:focus-visible{outline:3px solid rgba(59,130,246,.65);outline-offset:3px}" +
    ".pai-launcher-left{left:24px;right:auto}" +
    ".pai-frame{position:fixed;z-index:2147483646;right:24px;bottom:92px;width:min(380px,calc(100vw - 32px));height:min(640px,calc(100vh - 120px));border:1px solid rgba(15,23,42,.14);border-radius:8px;box-shadow:0 24px 70px rgba(15,23,42,.28);background:#fff;overflow:hidden;opacity:0;transform:translateY(8px);pointer-events:none;transition:opacity .18s ease,transform .18s ease}" +
    ".pai-frame-left{left:24px;right:auto}" +
    ".pai-open{opacity:1;transform:translateY(0);pointer-events:auto}" +
    "@media(max-width:480px){.pai-launcher{right:16px;bottom:16px}.pai-launcher-left{left:16px;right:auto}.pai-frame{inset:0;width:100vw;height:100vh;border:0;border-radius:0}.pai-frame-left{left:0}}";

  var button = document.createElement("button");
  button.type = "button";
  button.className = "pai-launcher" + (isLeft() ? " pai-launcher-left" : "");
  button.setAttribute("aria-label", "Open P&AI chat");
  button.innerHTML =
    '<svg aria-hidden="true" width="26" height="26" viewBox="0 0 24 24" fill="none"><path d="M5 6.75A3.75 3.75 0 0 1 8.75 3h6.5A3.75 3.75 0 0 1 19 6.75v4.5A3.75 3.75 0 0 1 15.25 15H11l-4.8 4.2A.75.75 0 0 1 5 18.64V15.1A3.75 3.75 0 0 1 2 11.25v-4.5Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/></svg>';

  var frame = document.createElement("iframe");
  frame.className = "pai-frame" + (isLeft() ? " pai-frame-left" : "");
  frame.title = clean(dataset.title, "P&AI Tutor");
  frame.src = widgetURL();
  frame.allow = "clipboard-write";
  frame.loading = "lazy";
  frame.referrerPolicy = "strict-origin-when-cross-origin";

  function open() {
    state.open = true;
    frame.classList.add("pai-open");
    button.setAttribute("aria-label", "Close P&AI chat");
    emit("open");
  }

  function close() {
    state.open = false;
    frame.classList.remove("pai-open");
    button.setAttribute("aria-label", "Open P&AI chat");
    emit("close");
  }

  function toggle() {
    if (state.open) {
      close();
      return;
    }
    open();
  }

  function destroy() {
    window.removeEventListener("message", handleMessage);
    root.remove();
    delete window.PAIWidget;
  }

  function handleMessage(event) {
    if (event.origin !== origin || !event.data || typeof event.data.type !== "string") {
      return;
    }

    if (event.data.type === "pai-widget:ready") {
      state.ready = true;
      emit("ready");
    }

    if (event.data.type === "pai-widget:close") {
      close();
    }
  }

  button.addEventListener("click", toggle);
  window.addEventListener("message", handleMessage);
  shadow.appendChild(style);
  shadow.appendChild(frame);
  shadow.appendChild(button);

  window.PAIWidget = {
    open: open,
    close: close,
    toggle: toggle,
    destroy: destroy,
  };
})();
