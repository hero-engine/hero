---
name: equivalence-partitioning
description: Select representative valid and invalid input classes without multiplying redundant cases.
---
# Equivalence partitioning

Define the input domain and the behavior expected for each class. Separate valid,
invalid, absent, malformed, unauthorized, and context-dependent classes when their
outcomes differ. Pick one representative per stable class, then add representatives
only where risk or implementation structure suggests different behavior. Record
the partition rationale in the case so future authors do not recreate equivalent
tests under different titles.

