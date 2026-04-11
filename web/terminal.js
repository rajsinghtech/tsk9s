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
  term.focus();
  window.addEventListener('resize', function() { fitAddon.fit(); });

  var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  var ws = new WebSocket(proto + '//' + location.host + '/ws');
  ws.binaryType = 'arraybuffer';

  ws.onopen = function() {
    ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
    term.onData(function(data) { ws.send(new TextEncoder().encode(data)); });
    term.onResize(function(size) {
      ws.send(JSON.stringify({ type: 'resize', cols: size.cols, rows: size.rows }));
    });
  };

  ws.onmessage = function(e) { term.write(new Uint8Array(e.data)); };

  ws.onclose = function() {
    term.write('\r\n\x1b[31mConnection closed. Refresh to reconnect.\x1b[0m\r\n');
  };
})();
