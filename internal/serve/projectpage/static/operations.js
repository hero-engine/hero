// Project section page — Operations row wiring.
//
// Click handler POSTs /api/{slug}/ops/{verb} then opens an EventSource
// on /api/{slug}/ops/{job_id}/stream. Progress lines append to the
// inline <pre>; the row's button re-enables on the exit event.
//
// Already-running verbs (server-rendered with data-active-job set) get
// an immediate EventSource attach on page load so a navigate-away /
// navigate-back never loses the stream.
(function () {
  var section = document.querySelector('section[data-section="operations"]');
  if (!section) return;
  var slug = section.getAttribute("data-project-slug") || "";
  if (!slug) return;

  var rows = section.querySelectorAll("li.project-op");
  rows.forEach(function (row) {
    var verb = row.getAttribute("data-verb") || "";
    var initialJob = row.getAttribute("data-active-job") || "";
    var btn = row.querySelector("button.project-op-run");
    var output = row.querySelector("pre.project-op-output");
    var status = row.querySelector(".project-op-status");

    function appendOutput(text) {
      if (!output) return;
      if (output.hasAttribute("hidden")) output.removeAttribute("hidden");
      output.textContent += text + "\n";
      output.scrollTop = output.scrollHeight;
    }

    function markRunning() {
      if (btn) {
        btn.setAttribute("disabled", "");
        btn.setAttribute("aria-busy", "true");
      }
      row.classList.add("op-in-flight");
      row.classList.remove("op-error");
      if (status) status.textContent = "running…";
    }

    function markDone(code, stderr) {
      if (btn) {
        btn.removeAttribute("disabled");
        btn.removeAttribute("aria-busy");
      }
      row.classList.remove("op-in-flight");
      if (code === 0) {
        if (status) status.textContent = "done";
      } else {
        row.classList.add("op-error");
        if (status) status.textContent = "failed (exit " + code + ")";
        if (stderr) appendOutput("stderr: " + stderr);
      }
    }

    function openStream(jobID) {
      markRunning();
      var url = "/api/" + encodeURIComponent(slug) + "/ops/" + encodeURIComponent(jobID) + "/stream";
      var es = new EventSource(url);
      es.addEventListener("progress", function (ev) {
        try {
          var f = JSON.parse(ev.data);
          if (f && typeof f.text === "string") appendOutput(f.text);
        } catch (e) {}
      });
      es.addEventListener("exit", function (ev) {
        var code = -1;
        var stderr = "";
        try {
          var f = JSON.parse(ev.data);
          code = typeof f.code === "number" ? f.code : -1;
          stderr = f.stderr || "";
        } catch (e) {}
        markDone(code, stderr);
        es.close();
      });
      es.addEventListener("error", function () {
        // The stream errors when the server side closes the response,
        // which happens right after the exit frame. Only treat this as
        // a real error when we never received an exit event — leave the
        // row in its current state otherwise.
        if (es.readyState === EventSource.CLOSED) return;
      });
    }

    if (initialJob) {
      openStream(initialJob);
    }

    if (btn) {
      btn.addEventListener("click", function () {
        if (btn.hasAttribute("disabled")) return;
        markRunning();
        if (output) {
          output.textContent = "";
          output.setAttribute("hidden", "");
        }
        fetch("/api/" + encodeURIComponent(slug) + "/ops/" + encodeURIComponent(verb), {
          method: "POST",
        })
          .then(function (resp) {
            if (!resp.ok) {
              return resp.text().then(function (body) {
                throw new Error("HTTP " + resp.status + ": " + body);
              });
            }
            return resp.json();
          })
          .then(function (body) {
            if (!body || !body.job_id) throw new Error("missing job_id in response");
            openStream(body.job_id);
          })
          .catch(function (err) {
            row.classList.remove("op-in-flight");
            row.classList.add("op-error");
            if (btn) {
              btn.removeAttribute("disabled");
              btn.removeAttribute("aria-busy");
            }
            if (status) status.textContent = "error";
            appendOutput("" + err);
          });
      });
    }
  });
})();
