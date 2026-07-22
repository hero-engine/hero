Pre-Req:

1. Provision a **stopped** VM.
2. Open [Discover](https://example.atlassian.net/browse/MORPH-297) and run `hero scan`.
    - Select all VMs.

Steps:

1. Click Bulk Start.
2. **Expected:** Every selected VM starts.<br>
   **Actual:** None starts — [FAILED] 🚨

Logs:

```bash
hero start --all
``unexpected``
exit 1
```

Preserved extension text.

### Notes

> This fails every time.

---
