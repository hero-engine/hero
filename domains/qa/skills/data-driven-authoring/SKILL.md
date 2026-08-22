---
name: data-driven-authoring
description: Design parameterized cases whose data rows represent meaningful behavior partitions rather than duplicated scripts.
---
# Data-driven authoring

Keep behavior and procedure fixed while varying only inputs and expected outputs.
Name each parameter, its type, source, sensitivity, and partition. Include a row
only when it exercises a distinct class, boundary, permission, locale, or expected
outcome. Make expected values explicit per row and avoid computed expectations that
repeat the product algorithm. Keep credentials and personal data out of committed
tables; reference safe fixtures instead.

