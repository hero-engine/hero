// project_all.js — cross-project /p/all/project page behaviour.
//
// Three small features, no framework:
//   1. Click a Directory table header to sort by that column. Click
//      again to reverse. Third click resets to original (DOM) order.
//   2. Type in the filter input to filter rows by slug / name / path.
//   3. Click "Refresh registry" to POST /api/daemon/registry/refresh,
//      then re-render the directory tbody from the JSON response.
(function () {
  var table = document.getElementById('project-directory-table');
  var filterInput = document.getElementById('project-directory-filter');
  var refreshBtn = document.getElementById('project-all-refresh');

  if (table) {
    var tbody = table.querySelector('tbody');
    var headers = table.querySelectorAll('thead th[data-sort-key]');
    // Snapshot original row order so a third click on the same column
    // can restore it.
    var originalRows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
    var sortState = { key: null, dir: 0 }; // dir: 0 reset, 1 asc, -1 desc

    headers.forEach(function (th, idx) {
      th.style.cursor = 'pointer';
      th.addEventListener('click', function () {
        var key = th.getAttribute('data-sort-key');
        if (sortState.key !== key) {
          sortState.key = key;
          sortState.dir = 1;
        } else {
          sortState.dir = sortState.dir === 1 ? -1 : sortState.dir === -1 ? 0 : 1;
        }
        if (sortState.dir === 0) {
          originalRows.forEach(function (r) { tbody.appendChild(r); });
          return;
        }
        var rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
        rows.sort(function (a, b) {
          var av = cellSortValue(a, idx);
          var bv = cellSortValue(b, idx);
          if (av === bv) return 0;
          // case-insensitive when both strings
          if (typeof av === 'string' && typeof bv === 'string') {
            return av.toLowerCase().localeCompare(bv.toLowerCase()) * sortState.dir;
          }
          return (av < bv ? -1 : 1) * sortState.dir;
        });
        rows.forEach(function (r) { tbody.appendChild(r); });
      });
    });

    function cellSortValue(row, idx) {
      var cells = row.children;
      if (idx >= cells.length) return '';
      var cell = cells[idx];
      var raw = cell.getAttribute('data-sort-value');
      if (raw !== null) {
        var n = Number(raw);
        if (!isNaN(n)) return n;
        return raw;
      }
      return (cell.textContent || '').trim();
    }
  }

  if (filterInput && table) {
    filterInput.addEventListener('input', function () {
      var q = filterInput.value.toLowerCase().trim();
      var rows = table.querySelectorAll('tbody tr');
      rows.forEach(function (row) {
        if (!q) {
          row.style.display = '';
          return;
        }
        var text = (row.textContent || '').toLowerCase();
        row.style.display = text.indexOf(q) >= 0 ? '' : 'none';
      });
    });
  }

  if (refreshBtn) {
    refreshBtn.addEventListener('click', function () {
      refreshBtn.disabled = true;
      var prevLabel = refreshBtn.textContent;
      refreshBtn.textContent = 'refreshing…';
      fetch('/api/daemon/registry/refresh', { method: 'POST' })
        .then(function (resp) {
          if (!resp.ok) throw new Error('http ' + resp.status);
          return resp.json();
        })
        .then(function () {
          // Simplest reliable path: reload the page so the server-
          // rendered rows are guaranteed to match the new registry.
          // Client-side row re-render is a follow-up if/when SSE for
          // the registry lands.
          window.location.reload();
        })
        .catch(function (err) {
          refreshBtn.disabled = false;
          refreshBtn.textContent = prevLabel;
          console.error('refresh registry failed', err);
        });
    });
  }

  // ---- Phase 4: Stop daemon (aggregate-only) ----
  //
  // Behavior: POST /api/daemon/ops/stop, then replace the page body
  // with a "Daemon stopped — relaunch with `hero serve`" inline message.
  // The daemon will die mid-request so the fetch usually errors with
  // a connection failure; we treat that as success and render the same
  // landing message. This is the "inline message" branch flagged back
  // to the spec author.
  var stopBtn = document.getElementById('project-stop-daemon-btn');
  var stopStatus = document.getElementById('project-stop-daemon-status');
  if (stopBtn) {
    stopBtn.addEventListener('click', function () {
      stopBtn.disabled = true;
      if (stopStatus) stopStatus.textContent = 'stopping…';
      function renderStopped() {
        document.body.innerHTML =
          '<main class="hero-stopped-landing" style="padding:48px 32px;font-family:system-ui,sans-serif;">' +
          '<h1>Daemon stopped</h1>' +
          '<p>The hero daemon is no longer running. Relaunch with <code>hero serve</code> and reload this page to continue.</p>' +
          '</main>';
      }
      fetch('/api/daemon/ops/stop', { method: 'POST' })
        .then(function (resp) {
          // Success path: daemon accepted the dispatch and the subprocess
          // is signalling. Poll briefly until the daemon stops responding.
          if (!resp.ok) throw new Error('http ' + resp.status);
          var attempts = 0;
          var probe = setInterval(function () {
            attempts += 1;
            fetch('/health')
              .then(function () {
                if (attempts >= 10) {
                  clearInterval(probe);
                  renderStopped();
                }
              })
              .catch(function () {
                clearInterval(probe);
                renderStopped();
              });
          }, 500);
        })
        .catch(function () {
          // Connection lost mid-request: daemon already died. Render
          // the stopped landing immediately.
          renderStopped();
        });
    });
  }
})();
