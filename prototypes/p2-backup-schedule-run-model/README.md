# P2 backup schedule-to-run identity model (synthetic only)

This standard-library model exercises one proposed P2 acceptance property:
repeated delivery of the same synthetic scheduled-backup occurrence maps to one
immutable, attributable run record. Schedule ownership comes from an in-memory
fixture registration. A logical key is `(registered tenant, schedule ID,
occurrence token)`; the token is an opaque test input, not a production ID or a
clock-derived recurrence value.

The tests cover first delivery, exact duplicate delivery returning the same
record, reuse of one occurrence token across different tenants' schedules,
generic denial for unknown schedule references and any event-supplied tenant
override (including a value matching the registered owner), conflicting
revision replay, immutable run and schedule records, distinct occurrences,
duplicate fixture registrations, and malformed identifiers with no state
mutation. Tenant ownership comes only from fixture registration; the delivery
event has no tenant field. The denied-override case is a fixture contract check,
not a cross-tenant authorization decision.

This model does not implement or choose a scheduler, schedule cadence, cron
grammar, timezone or daylight-saving behavior, queue or backlog policy,
authorization, durable storage or restart recovery, execution, retries, transfer,
encryption, retention, restore, or actor/audit policy. Tenant mismatch checks
are fixture identity checks, not an authorization implementation. A `RECORDED`
run means only that one logical run record exists; it does not show that a backup
executed or succeeded. The model is acceptance preparation only. It does not
close issue #152, satisfy P2 acceptance, or change owner/security gates.

Run the focused tests from the repository root with:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s prototypes/p2-backup-schedule-run-model -p 'test_*.py' -v
```
