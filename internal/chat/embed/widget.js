(function() {
  'use strict';

  var script = document.currentScript;
  var tenant = script.getAttribute('data-tenant') || '';
  var rawColor = script.getAttribute('data-color') || '#b45a1a';
  var color = /^#[0-9a-fA-F]{3,8}$/.test(rawColor) ? rawColor : '#b45a1a';
  var position = script.getAttribute('data-position') || 'bottom-right';
  var lang = script.getAttribute('data-language') || '';

  // Derive the base URL from the script's own src.
  var src = script.src;
  var baseURL = src.substring(0, src.lastIndexOf('/embed/'));

  // Create toggle button (chat bubble).
  var btn = document.createElement('div');
  btn.id = 'pai-chat-toggle';
  btn.innerHTML = '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';
  btn.style.cssText = 'position:fixed;' + (position === 'bottom-left' ? 'left:20px;' : 'right:20px;') + 'bottom:20px;width:54px;height:54px;border-radius:14px;background:' + color + ';color:#fef5e7;display:flex;align-items:center;justify-content:center;cursor:pointer;box-shadow:0 4px 16px rgba(0,0,0,0.12),0 1px 3px rgba(0,0,0,0.08);z-index:99998;border:none;transition:transform 0.2s ease,box-shadow 0.2s ease;';

  btn.addEventListener('mouseenter', function() { btn.style.transform = 'scale(1.06)'; btn.style.boxShadow = '0 6px 20px rgba(0,0,0,0.16),0 2px 4px rgba(0,0,0,0.1)'; });
  btn.addEventListener('mouseleave', function() { btn.style.transform = 'scale(1)'; btn.style.boxShadow = '0 4px 16px rgba(0,0,0,0.12),0 1px 3px rgba(0,0,0,0.08)'; });

  // Create iframe container.
  var container = document.createElement('div');
  container.id = 'pai-chat-container';
  container.style.cssText = 'position:fixed;' + (position === 'bottom-left' ? 'left:20px;' : 'right:20px;') + 'bottom:86px;width:380px;height:520px;max-height:80vh;border-radius:14px;overflow:hidden;box-shadow:0 8px 30px rgba(0,0,0,0.12),0 2px 8px rgba(0,0,0,0.06);z-index:99999;display:none;border:1px solid rgba(0,0,0,0.08);';

  var iframe = document.createElement('iframe');
  iframe.src = baseURL + '/embed/chat?tenant=' + encodeURIComponent(tenant) + '&color=' + encodeURIComponent(color) + '&lang=' + encodeURIComponent(lang);
  iframe.style.cssText = 'width:100%;height:100%;border:none;';
  iframe.setAttribute('sandbox', 'allow-scripts allow-same-origin allow-forms');
  iframe.setAttribute('title', 'P&AI Chat');

  container.appendChild(iframe);
  document.body.appendChild(container);
  document.body.appendChild(btn);

  var open = false;
  btn.addEventListener('click', function() {
    open = !open;
    container.style.display = open ? 'block' : 'none';
    btn.innerHTML = open
      ? '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>'
      : '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';
  });
})();
