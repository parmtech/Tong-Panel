# P1 CI investigation handoff

Draft checkpoint only; not a security approval or release determination.

On prior PR head 75d6e9a9c0afbff5301a2a23df097a5a686099c0, the pinned gosec scanner produced JSON with `Golang errors`, while the validator expected `Errors`. The downloaded report contained 241 findings and an empty analyzer-error object. The validator now checks the actual field and still rejects analyzer errors and missing fields. Findings are not suppressed or resolved by this change; they require review. The workflow uses `-no-fail`, so a successful report-validation job does not mean no findings.

The vulnerability scans reported GO-2026-6859 in Go 1.27.3, fixed in Go 1.27.4. The two affected scan workflows now select Go 1.27.4; hosted runs on the new head must confirm results. This does not establish that other vulnerabilities are absent.

Local validation: the prior downloaded report passed the revised validator; synthetic analyzer-error and missing-field inputs were rejected. Other prior-head tests do not prove the new hosted checks pass. Review the new workflow results, assess all scanner findings, and complete the outstanding owner and security decisions before considering phase acceptance.

Hosted verification update: on commit `c1f672a875f830c72daee60c444fd472afaed53f`, nine PR workflows succeeded, including gosec report validation, both vulnerability scans, API contract, Go checks, UI, jobs browser, developer smoke, unsigned package lab, and P2 disposable VM smoke. These results apply to that commit; the 241 reported findings still require triage and are not resolved by a passing report-validation job. Subsequent documentation commits are distinct heads.

Hosted scan follow-up: commit `0ba88419036b4c923e70242d496c462b93515d42` failed the gosec report-validation job because gosec attempted to load the nested ACME module requiring Go 1.27.4 while the scanner workflow used Go 1.27.3 with GOTOOLCHAIN=local. The scanner workflow now uses Go 1.27.4; analyzer errors remain fatal. The new workflow run must be checked before claiming success.
