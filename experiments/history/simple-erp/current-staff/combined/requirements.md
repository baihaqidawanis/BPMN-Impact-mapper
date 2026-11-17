# combined

Require submit reason and both active-master validations. Any failure must not submit; existing state/access/version guards remain.

Submit currently only checks DRAFT/version. Reason field exists but UI submits empty reason. Master active checks during create do not ensure they remain active at submit. Operations integration tests exercise existing submit/guards. No schema or new endpoint required.

Reference scope: one selected primary change; other changes not labelled or counted as negatives.
Reference status: provisional source-derived diagnostic; prepared before inference, not independent ground truth.
