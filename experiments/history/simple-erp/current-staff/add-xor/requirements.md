# add-xor

Require non-empty reason before adjustment submit.

Submit currently only checks DRAFT/version. Reason field exists but UI submits empty reason. Master active checks during create do not ensure they remain active at submit. Operations integration tests exercise existing submit/guards. No schema or new endpoint required.

Reference scope: one selected primary change; other changes not labelled or counted as negatives.
Reference status: provisional source-derived diagnostic; prepared before inference, not independent ground truth.
