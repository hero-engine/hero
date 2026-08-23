# Build information

This tracked page is a build-time template. Its placeholder values make a local
source preview structurally valid without pretending to identify a deployed
artifact. The documentation workflow replaces them in its ephemeral workspace
before the strict build and deployment.

| Field | Value |
|---|---|
| Current released version | `BUILD_TIME_CURRENT_RELEASE` |
| Documentation source revision | `BUILD_TIME_SOURCE_REVISION` |
| Artifact generated at | `BUILD_TIME_GENERATED_AT` |

Machine-readable marker: [`/revision.json`](../revision.json).

In a built artifact, the release is derived from the latest reachable Git tag,
the revision is the checked-out commit, and the timestamp records artifact
generation. A local strict build proves source consistency only. Deployed
parity is established by reading the generated values from production after
deployment. Placeholder values mean no deployment identity is being claimed.
