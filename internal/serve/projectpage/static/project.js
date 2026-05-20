// Project section page — collapse/expand wiring.
//
// This file mirrors the JS inlined by projectpage/handler.go's
// projectScript helper. It is committed here as a reference / future
// extraction point: Phase 1 inlines the script so we don't fan out a
// new /static/ route for a single tiny file. Phase 5 will extract this
// into a real static asset when section-refresh wiring lands.
(function () {
  // slug is interpolated by the handler when this file is eventually
  // served. The reference copy uses a placeholder.
  var slug = window.__heroProjectSlug || "";
  var heads = document.querySelectorAll(".project-section-head");
  heads.forEach(function (head) {
    var section = head.closest(".project-section");
    if (!section) return;
    var name = section.getAttribute("data-section") || "";
    var key = "hero-projectpage:" + slug + ":" + name;
    var body = section.querySelector(".project-section-body");
    var btn = head.querySelector(".project-section-toggle");
    var defaultCollapsed = head.getAttribute("data-default-collapsed") === "true";
    var stored = null;
    try {
      stored = localStorage.getItem(key);
    } catch (e) {}
    var collapsed = stored === null ? defaultCollapsed : stored === "1";
    apply();
    if (btn) {
      btn.addEventListener("click", function () {
        collapsed = !collapsed;
        try {
          localStorage.setItem(key, collapsed ? "1" : "0");
        } catch (e) {}
        apply();
      });
    }
    function apply() {
      if (!body) return;
      if (collapsed) {
        body.setAttribute("hidden", "");
      } else {
        body.removeAttribute("hidden");
      }
      if (btn) {
        btn.setAttribute("aria-expanded", collapsed ? "false" : "true");
        btn.textContent = collapsed ? "expand" : "collapse";
      }
    }
  });
})();
