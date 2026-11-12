# P1 CI investigation handoff

Draft checkpoint only; not a security approval or release determination.

On prior PR head 75d6e9a9c0afbff5301a2a23df097a5a686099c0, the pinned gosec scanner produced JSON with `Golang errors`, while the validator expected `Errors`. The downloaded report contained 241 findings and an empty analyzer-error object. The validator now checks the actual field and still rejects analyzer errors and missing fields. Findings are not suppressed or resolved by this change; they require review. The workflow uses `-no-fail`, so a successful report-validation job does not mean no findings.

The vulnerability scans reported GO-2026-6859 in Go 1.27.3, fixed in Go 1.27.4. The two affected scan workflows now select Go 1.27.4; hosted runs on the new head must confirm results. This does not establish that other vulnerabilities are absent.

Local validation: the prior downloaded report passed the revised validator; synthetic analyzer-error and missing-field inputs were rejected. Other prior-head tests do not prove the new hosted checks pass. Review the new workflow results, assess all scanner findings, and complete the outstanding owner and security decisions before considering phase acceptance.
