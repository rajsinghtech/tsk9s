(function() {
  var term = new Terminal({
    fontSize: 14,
    theme: { background: '#1a1b26', foreground: '#c0caf5' },
    cursorBlink: true,
    allowTransparency: false,
  });
  var fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  term.open(document.getElementById('terminal'));
  fitAddon.fit();
  window.addEventListener('resize', function() { fitAddon.fit(); });

  var ws = null;
  var activeBtn = null;

  function connect(cluster, btn) {
    if (ws) { ws.close(); ws = null; }

    document.getElementById('placeholder').classList.add('hidden');

    if (activeBtn) activeBtn.classList.remove('active');
    activeBtn = btn;
    btn.classList.add('active');

    term.clear();
    term.focus();

    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(proto + '//' + location.host + '/ws?cluster=' + encodeURIComponent(cluster));
    ws.binaryType = 'arraybuffer';

    ws.onopen = function() {
      ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
      term.onData(function(data) { if (ws && ws.readyState === 1) ws.send(new TextEncoder().encode(data)); });
      term.onResize(function(size) {
        if (ws && ws.readyState === 1) ws.send(JSON.stringify({ type: 'resize', cols: size.cols, rows: size.rows }));
      });
    };

    ws.onmessage = function(e) { term.write(new Uint8Array(e.data)); };

    ws.onclose = function() {
      btn.classList.remove('active');
      activeBtn = null;
      term.write('\r\n\x1b[31mConnection closed. Click a cluster to reconnect.\x1b[0m\r\n');
    };
  }

  fetch('/api/clusters')
    .then(function(r) { return r.json(); })
    .then(function(clusters) {
      var list = document.getElementById('cluster-list');
      (clusters || []).forEach(function(c) {
        var btn = document.createElement('button');
        btn.className = 'cluster-btn';
        btn.textContent = c.Name;
        btn.title = c.FQDN;
        btn.onclick = function() { connect(c.Name, btn); };
        list.appendChild(btn);
      });
      // Auto-connect to first cluster
      if (clusters && clusters.length > 0) {
        var first = list.querySelector('.cluster-btn');
        if (first) first.click();
      }
    });
})();
