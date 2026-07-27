(function() {
  'use strict';

  if (window.__paiChatLoading || document.getElementById('pai-chat-toggle') || document.getElementById('pai-chat-container')) {
    return;
  }

  var script = document.currentScript;
  if (!script) return;
  var tenant = script.getAttribute('data-tenant') || '';
  var src = script.src;
  var baseURL = src.substring(0, src.lastIndexOf('/embed/'));
  var baseOrigin = new URL(baseURL).origin;
  var parentOrigin = window.location.origin;
  window.__paiChatLoading = true;

  var configURL = baseURL + '/api/embed/config?tenant=' + encodeURIComponent(tenant) + '&parent_origin=' + encodeURIComponent(parentOrigin);
  fetch(configURL, { credentials: 'omit', mode: 'cors' })
    .then(function(response) {
      if (!response.ok) throw new Error('embed unavailable');
      return response.json();
    })
    .then(function(config) {
      if (!config || config.enabled !== true) return;
      createWidget(config.theme_config || {});
    })
    .catch(function() {})
    .finally(function() { window.__paiChatLoading = false; });

  function createWidget(theme) {
    if (document.getElementById('pai-chat-toggle') || document.getElementById('pai-chat-container')) return;
    var rawColor = theme.color || script.getAttribute('data-color') || '#b45a1a';
    var color = /^#[0-9a-fA-F]{6}$/.test(rawColor) ? rawColor : '#b45a1a';
    var foreground = readableForeground(color);
    var position = theme.position === 'bottom-left' ? 'bottom-left' : 'bottom-right';
    var lang = theme.language === 'ms' || theme.language === 'zh' ? theme.language : 'en';
    var copy = {
      en: { open: 'Open P&AI chat', close: 'Close P&AI chat', title: 'P&AI Chat' },
      ms: { open: 'Buka sembang P&AI', close: 'Tutup sembang P&AI', title: 'Sembang P&AI' },
      zh: { open: '打开 P&AI 聊天', close: '关闭 P&AI 聊天', title: 'P&AI 聊天' }
    }[lang];

    var btn = document.createElement('button');
    btn.id = 'pai-chat-toggle';
    btn.type = 'button';
    btn.setAttribute('aria-label', copy.open);
    btn.setAttribute('aria-expanded', 'false');
    btn.setAttribute('aria-controls', 'pai-chat-container');
    btn.innerHTML = chatIcon();
    btn.style.cssText = 'position:fixed;' + (position === 'bottom-left' ? 'left:20px;' : 'right:20px;') + 'bottom:20px;width:54px;height:54px;border-radius:14px;background:' + color + ';color:' + foreground + ';display:flex;align-items:center;justify-content:center;cursor:pointer;box-shadow:0 4px 16px rgba(0,0,0,0.12),0 1px 3px rgba(0,0,0,0.08);z-index:99998;border:none;transition:transform 0.2s ease,box-shadow 0.2s ease;';

    btn.addEventListener('mouseenter', function() { btn.style.transform = 'scale(1.06)'; btn.style.boxShadow = '0 6px 20px rgba(0,0,0,0.16),0 2px 4px rgba(0,0,0,0.1)'; });
    btn.addEventListener('mouseleave', function() { btn.style.transform = 'scale(1)'; btn.style.boxShadow = '0 4px 16px rgba(0,0,0,0.12),0 1px 3px rgba(0,0,0,0.08)'; });

    var container = document.createElement('div');
    container.id = 'pai-chat-container';
    container.style.cssText = 'position:fixed;' + (position === 'bottom-left' ? 'left:20px;' : 'right:20px;') + 'bottom:86px;width:380px;max-width:calc(100vw - 40px);height:520px;max-height:calc(100vh - 106px);max-height:calc(100dvh - 106px);border-radius:14px;overflow:hidden;box-shadow:0 8px 30px rgba(0,0,0,0.12),0 2px 8px rgba(0,0,0,0.06);z-index:99999;display:none;border:1px solid rgba(0,0,0,0.08);background:#fff;';

    var iframe = document.createElement('iframe');
    iframe.src = baseURL + '/embed/chat?tenant=' + encodeURIComponent(tenant) + '&color=' + encodeURIComponent(color) + '&lang=' + encodeURIComponent(lang);
    iframe.style.cssText = 'width:100%;height:100%;border:none;';
    iframe.setAttribute('sandbox', 'allow-scripts allow-same-origin allow-forms');
    iframe.setAttribute('title', copy.title);
    iframe.addEventListener('load', function() {
      iframe.contentWindow.postMessage({
        type: 'pai-parent-origin',
        parent_origin: parentOrigin
      }, baseOrigin);
    });

    container.appendChild(iframe);
    document.body.appendChild(container);
    document.body.appendChild(btn);

    var open = false;
    btn.addEventListener('click', function() {
      open = !open;
      container.style.display = open ? 'block' : 'none';
      btn.setAttribute('aria-expanded', open ? 'true' : 'false');
      btn.setAttribute('aria-label', open ? copy.close : copy.open);
      btn.innerHTML = open ? closeIcon() : chatIcon();
    });
  }

  function readableForeground(color) {
    var values = [1, 3, 5].map(function(offset) {
      var value = parseInt(color.slice(offset, offset + 2), 16) / 255;
      return value <= 0.04045 ? value / 12.92 : Math.pow((value + 0.055) / 1.055, 2.4);
    });
    var luminance = 0.2126 * values[0] + 0.7152 * values[1] + 0.0722 * values[2];
    var darkLuminance = 0.0097;
    var darkContrast = (Math.max(luminance, darkLuminance) + 0.05) / (Math.min(luminance, darkLuminance) + 0.05);
    var lightContrast = 1.05 / (luminance + 0.05);
    return darkContrast >= lightContrast ? '#111827' : '#ffffff';
  }

  function chatIcon() {
    return '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';
  }

  function closeIcon() {
    return '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
  }
})();
