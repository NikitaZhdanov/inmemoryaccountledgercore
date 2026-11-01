# Numerical Inventory

This document records all numerical sources, explicit parameters, derived values, and precision bounds used in the ledger core.

## Constants and sources

| Parameter | Value | Source / Business Rationale |
| :--- | :--- | :--- |
| **Replay Window** | 6 days (Day 1 – Day 6) | Sequence bounds defined by task requirements. |
| **Accounts** | `ACC-001` (AED), `ACC-002` (BHD) | Opening balance for both is `0.00` / `0.000`. |
| **AED Overdraft Fee** | AED 25.00 | Assessed once per day on negative closing ledger balances. |
| **BHD Overdraft Fee** | BHD 0.000 | Explicit zero-fee rule (no fee tariff specified for BHD). |
| **Daily Interest Rate** | 0.04% ($0.0004$) per day | Accrued on positive closing ledger balances; capitalized on Day 6. |

The exact rate is $4/10,000 = 1/2,500$. Daily interest divides positive
minor units by 2,500; no balance-times-rate intermediate is needed.

---

## 2. Precision & Representation

### Int64 representation

Signed minor units range from −9,223,372,036,854,775,808 through
9,223,372,036,854,775,807. The asymmetric endpoints below correct the previous
table's symmetric bounds.

| Currency | Scale (Decimals) | Factor | Storage Format | Min / Max Bounds |
| :--- | :--- | :--- | :--- | :--- |
| **AED** | 2 | $10^2 = 100$ | Signed `int64` minor units | −92,233,720,368,547,758.08 through 92,233,720,368,547,758.07 AED |
| **BHD** | 3 | $10^3 = 1,000$ | Signed `int64` minor units | −9,223,372,036,854,775.808 through 9,223,372,036,854,775.807 BHD |

- **Arithmetic Safety:** All calculations use checked `int64` arithmetic. Operations that overflow or underflow the minor unit bounds are rejected with errors.
- **Input Validation:** Amounts exactly representable at currency scale are accepted, including redundant trailing zeros. Nonzero excess precision (e.g., AED `1.005`) is rejected; calculated interest rounding does not apply to principal.

At the maximum positive base, independent long division gives
9,223,372,036,854,775,807 = 2,500 × 3,689,348,814,741,910 + 807.
Since 807 < 1,250, daily interest is 3,689,348,814,741,910 minor units:
AED 36,893,488,147,419.10 or BHD 3,689,348,814,741.910.

---

## Allocation and rounding rules

- **Interest Accrual Rounding:** Calculated daily using `HALF_EVEN` (Banker's Rounding) to the currency's decimal precision.

### Verification parameters

For a positive base $m$ in minor units, let $q = m / 2,500$ and
$r = m \bmod 2,500$. Increment $q$ when $r > 1,250$, or when
$r = 1,250$ and $q$ is odd. These fixtures exercise each side of the
midpoint and both tie directions in AED and BHD; results are minor units.

| Base minor units | Expected interest minor units | Boundary |
| ---: | ---: | --- |
| −9,223,372,036,854,775,808; −1; 0 | 0 | Nonpositive base |
| 1; 1,249 | 0 | Below first midpoint |
| 1,250 | 0 | 0.5 → even zero |
| 1,251; 2,500; 3,749 | 1 | Above midpoint, exact unit, below next midpoint |
| 3,750; 3,751 | 2 | 1.5 → even two, then above midpoint |
| 6,250 | 2 | 2.5 → even two |

The midpoint bases are AED 12.50 / BHD 1.250 and AED 37.50 / BHD 3.750;
rounded results are respectively AED 0.00 / BHD 0.000 and AED 0.02 / BHD 0.002.
Replay calculation fixtures are 250.00 → 0.10, 650.00 → 0.26,
465.00 → 0.19 AED and 10.000 → 0.004 BHD, as in the final table below.

Shared ratio rounding uses signed numerator $a$ and positive denominator $d$.
With truncation-toward-zero quotient $q$ and remainder magnitude $r$, adjust $q$
away from zero when $r > d-r$, or when $r = d-r$ and $q$ is odd.
Comparing distances avoids doubling a potentially large remainder.

| Ratio fixtures | Rounded integers | Purpose |
| --- | --- | --- |
| 0/1; 1/1; −1/1 | 0; 1; −1 | Zero and exact signed integers |
| 1/2; 3/2; 5/2 | 0; 2; 2 | Positive even-tie directions |
| −1/2; −3/2; −5/2 | 0; −2; −2 | Negative even-tie directions |
| 1/3; 2/3; −1/3; −2/3 | 0; 1; 0; −1 | Odd denominator, both sides of midpoint |
| Either int64 endpoint / 1 | Original endpoint | Full signed range |
| 9,223,372,036,854,775,807 / 2 | 4,611,686,018,427,387,904 | Maximum numerator rounds up |
| −9,223,372,036,854,775,808 / 3 | −3,074,457,345,618,258,603 | Minimum numerator rounds away from zero |
| Either int64 endpoint / 9,223,372,036,854,775,807 | 1 or −1 respectively | Maximum denominator |
| 4,611,686,018,427,387,903 / 9,223,372,036,854,775,807 | 0 | Below large midpoint |
| 4,611,686,018,427,387,904 / 9,223,372,036,854,775,807 | 1 | Above large midpoint |

Denominators 0, −1, and −9,223,372,036,854,775,808 verify nonpositive-denominator
rejection. Signed rounding defines an arithmetic primitive; interest retains its
positive-base eligibility rule.

### Instalment conservation

For signed minor units $m$ and positive count $n$, integer division toward zero
gives $q$ and residual $r = m - nq$. Allocate $q$ to the first $n-1$ parts
and $q+r$ to the last; their exact sum is $m$. Counts zero and −1 verify
nonpositive-count rejection. Signed arithmetic fixtures specify the pure
calculation, without defining which future inputs may post credits or debits.

- **Instalment Allocation (E10):** Splitting BHD 10.000 across 3 equal payments produces quotient BHD 3.333 and a 1-minor-unit residual. Per policy, remainder is added to the **last instalment**:
  - Instalment 1: BHD 3.333
  - Instalment 2: BHD 3.333
  - Instalment 3: BHD 3.334 (Sum = BHD 10.000)

Additional allocation oracles apply at both currency scales:

| Total minor units | Count | Ordered minor-unit results | Purpose |
| ---: | ---: | --- | --- |
| 9,999 | 3 | 3,333; 3,333; 3,333 | Exact division |
| 10,001 | 3 | 3,333; 3,333; 3,335 | Full two-unit residual goes last |
| 2 | 3 | 0; 0; 2 | Total smaller than count |
| 0 | 3 | 0; 0; 0 | Zero principal |
| −10,000 | 3 | −3,333; −3,333; −3,334 | Signed conservation |
| 9,223,372,036,854,775,807 | 3 | 3,074,457,345,618,258,602; 3,074,457,345,618,258,602; 3,074,457,345,618,258,603 | Maximum total |
| −9,223,372,036,854,775,808 | 3 | −3,074,457,345,618,258,602; −3,074,457,345,618,258,602; −3,074,457,345,618,258,604 | Minimum total without negation |
| Either signed endpoint | 1 | Original total | Single-part identity |

---

## Checkpoint calculations

P04 history fixtures record principal, fee, and correction effects directly.
P05 additionally posts E1/E2/E4 through the core; P06 integration submits E1–E6.
P07 adds E7/E9 principal posting and E10 instalments. P08 generates the fee/refund
and closed-day accrual results below through explicit closing/reconciliation with
controlled principal fixtures; P09 verifies the same oracle through submissions
with real storage and explicit close checkpoints. P10 extends that integration
through finalization using the six-day oracle below. P11 invokes the reusable
scenario runner and independently checks the same checkpoints and final values.

The ten scenario inputs retain TASK's literal fields and stream positions 1–10.
Ordinary closes for Days 1–5 follow E2/E3/E4/E6/E8; Day 6 finalizes after E10,
as specified in [SPEC](docs/SPEC.md#history-and-replay). These dates and amounts
are supplied requirements; reducing them would change the replay. Resubmission
adds 10 × 3 = 30 attempt/outcome/diagnostic records without financial or hold
changes. Descriptor-injection fixtures use XYZ/4 and ABC/18 to distinguish
configured values from canonical currency defaults; neither sets a core policy.

### ACC-001 checkpoints (AED)

| Observation | Value day | Fee-free balance | Posted balance |
| --- | ---: | ---: | ---: |
| E1/E2 | 1 | 1,200.00 − 950.00 = 250.00 | 250.00 |
| E4 | 3 | 250.00 + 400.00 = 650.00 | 650.00 |
| E5 | 4 | 650.00 − 185.00 = 465.00 | 465.00 |
| E7 through Day 5 close | 2 | 250.00 − 620.00 = −370.00 | −370.00 − 25.00 = −395.00 |
| E7 through Day 5 close | 3 | 650.00 − 620.00 = 30.00 | 30.00 − 25.00 = 5.00 |
| E7 through Day 5 close | 4 | 465.00 − 620.00 = −155.00 | −155.00 − 50.00 = −205.00 |
| E7 through Day 5 close | 5 | −155.00 | −155.00 − 75.00 = −230.00 |
| E9 and linked refunds | 2 | 250.00 | 250.00 |
| E9 and linked refunds | 5 | 465.00 | 465.00 |

Fee obligations for Days 2, 4, and 5 each have signed net −25.00 before E9;
each linked +25.00 refund restores its net to zero. A renewed −25.00 adjustment
under the same obligation restores net −25.00. Own-day fee exclusion removes
every assessment/refund/adjustment for that day, retaining earlier fees: Day 4
eligibility is −180.00; Day 5 eligibility before its fee is −205.00. Excluding
all fees gives −155.00 on both days.

Auth-A's 200.00 hold leaves availability 250.00 − 200.00 = 50.00 at Day 2
and 650.00 − 200.00 = 450.00 at Day 3. Settlement releases the entire hold;
ledger and availability both become 465.00. At E8, before Day 5's fee,
availability is −205.00; the requested 90.00 would give −295.00. Its recorded
decline remains after E9, without a hold.

AED daily accruals for Days 1–5 after E7/Day 5 close are 0.10, 0.00, 0.00,
0.00, 0.00. The Day 3 base 5.00 gives unrounded 0.002, hence 0.00.
E9 restores 0.10, 0.10, 0.26, 0.19, 0.19. E10 leaves AED unchanged.

### ACC-002 checkpoints (BHD)

E10 is processed after E9 while retaining booking/value Day 5. Its three
instalments total BHD 10.000 and replace the already-closed Day 5 accrual.
No fees or holds affect this account in the supplied replay.

| Observation | Value day | Posted balance (BHD) | Daily accrual (BHD) |
| --- | ---: | ---: | ---: |
| Before E10, including after E9 | 1–5 | 0.000 | 0.000 |
| After E10 and historical reconciliation | 1–4 | 0.000 | 0.000 |
| After E10 and historical reconciliation | 5 | 10.000 | 0.004 |
| Day 6 finalization | 6 | 10.008 | 0.004 |

The Day 6 close includes the single BHD 0.008 capitalization credit:
0.004 + 0.004 from Days 5 and 6. The daily balances and accruals for both
accounts are tabulated in [expected replay results](#expected-replay-results).

### Report verification fixtures

- Six supplied days × two accounts produce 12 rows, ordered by day then account.
  Currency scales, opening amounts, and policies come from the constants above;
  reducing the days/accounts or halving amounts would change the assigned replay.
- Reordered Days 2/1 and accounts Y/X verify caller-supplied ordering. XYZ/4 at
  one minor unit and ABC/0 at negative one minor unit verify precision without
  built-in currencies; these are the smallest nonzero representable values.
- A one-unit XYZ fee, refund, and renewed charge share assessed Day 1 while booked
  Days 2/3/4. Net balance is zero from a one-unit opening. Authorization IDs Z/A
  verify recorded order rather than alphabetical sorting; approval on Day 1 and
  settlement on Day 2 distinguish lifecycle from the final state. Reused event E
  has distinct declined/rejected attempts on Days 1/2 to verify diagnostic links.
- Failure fixtures reuse MaxInt64 plus one unit, missing-account lookup, and an
  unclosed Day 0; they require an error with no partial report. The integration
  precision rejection uses the existing AED `1.005` fixture without financial effects.

### Projection boundary fixtures

- One minor unit of opening money plus one future-value credit distinguishes
  opening balance from value-date inclusion. Two active holds of one unit each
  exactly exhaust the two-unit balance. These are minimal nonzero arithmetic
  fixtures, not additional business amounts.
- A separate AED base 37.49 yields rounded daily interest 0.01. Capitalizing
  two days' total 0.02 on the second day gives posted balance 37.51, while
  that day's interest remains 0.01. On the following day, 37.51 yields 0.02.
  These values straddle the 37.50 tie to expose same-day capitalization leakage.
- Balance accumulation `MaxInt64 + 1 − 1` fails at the intermediate sum even
  though its mathematical total fits. Fee net `MinInt64 − 1`, active holds
  `MaxInt64 + 1`, and availability `MinInt64 − 1` also fail explicitly. One
  minor unit is the smallest representable boundary crossing.
- An observation prefix counts records: zero is empty and full length includes
  all records. Prefixes −1 and length + 1 reject as the nearest invalid bounds.

### Ordinary submission fixtures

- Nonnegative CREDIT/DEBIT amounts include zero; DEBIT negates the parsed
  amount. Negative AED `-0.01` and the representable minimum reject under
  [A10](AMBIGUITIES.md#a10--ordinary-input-amount-signs); textual `-0.00` is zero.
- AED `1.000` stores 100 minor units; BHD `1.0000` stores 1,000. AED `1.005`
  and BHD `1.0001` reject excess precision. Decimal exponent text `1e2` is
  outside the fixed-point input grammar.
- AED `92233720368547758.08` and `-92233720368547758.09` lie one minor unit
  outside the signed range and reject before sign-policy validation. The
  positive endpoint posts exactly as a credit or negated debit; a further
  one-unit debit reaches the negative endpoint exactly.
- Opening `MaxInt64` plus a one-unit credit and opening `MinInt64` minus a
  one-unit debit overflow. A one-unit backdated credit also overflows when a
  later-value credit already posts `MaxInt64`, despite fitting on its own day.
- Configurable-scale submissions retain exact trailing zeros: scale zero
  `1.000`, scale four `0.00010`, and scale 18 `0.0000000000000000010` each
  post one minor unit. `1.1`, `0.00001`, and `0.0000000000000000001`
  respectively exceed those scales.
- Initialization uses canonical charges 2,500 AED / zero BHD minor units;
  −1 rejects a negative policy. Changing an external zero policy to one unit
  tests configuration ownership. One unit is the smallest representable change.
- Committed attempt identities count upward from one, independently of source
  stream positions. An ordinary accepted or rejected attempt contributes three
  records: input, outcome, then posting or diagnostic.

### Authorization and settlement fixtures

- Under [resolved A15](AMBIGUITIES.md#a15--authorization-and-settlement-amount-signs),
  AED `0.00` and textual `-0.00` pass amount validation for AUTHORIZATION and
  SETTLEMENT. A zero hold leaves AED 250.00 unchanged; a zero final settlement
  releases an existing AED 200.00 hold without changing ledger balance.
  A zero hold request on AED −0.01 still declines for insufficient availability.
- AED `-0.01` (the smallest negative amount) and the representable minimum
  `-92233720368547758.08` reject with `DiagnosticNegativeAmount` for both kinds,
  preserving balances and existing holds. AUTHORIZATION reserves a nonnegative
  amount; SETTLEMENT posts its checked negation, so input signs cannot create
  negative holds or settlement credits.
- E3 reserves AED 200.00 from AED 250.00, leaving AED 50.00 available. A second
  AED 50.00 hold leaves exactly zero and approves; a further AED 0.01 request
  declines. AED 0.01 is the smallest representable boundary crossing.
- A posted AED 25.00 fee reduces AED 250.00 to AED 225.00; its daily interest
  rounds to AED 0.09. A request of AED 225.01 declines even with that unpaid
  accrual and a future-value AED 0.01 credit; AED 225.00 approves exactly.
- E5 releases all AED 200.00 and debits AED 185.00, leaving AED 465.00.
  A separate active AED 50.00 hold remains reserved in the multiple-hold fixture,
  giving AED 415.00 available. E6's AED 180.00 request changes neither balance.
  A fresh AED 0.01 credit after E6 gives AED 465.01, testing continuation.
- The Auth-B correction fixture and P09 integration use
  AED 465.00 − 620.00 − 25.00 − 25.00 = −205.00 before the open Day 5's fee.
  Its AED 90.00 request would give
  −295.00; restoring principal and both fees returns AED 465.00 while retaining
  the decline. P09 generates the fees before E8 and closes Day 5 afterward.
- Proposed availability `MinInt64 − 1` fails before append. Separately,
  `25,000 + MinInt64 − 24,999 = MinInt64 + 1` at a future value date fits;
  reserving two minor units at the earlier AED 250.00 checkpoint would underflow
  future availability and appends nothing.
- Approval appends three records (attempt, outcome, transition); decline appends
  four with its diagnostic. Final settlement appends four (attempt, outcome,
  debit, transition); unknown-ID rejection appends three without financial effects.

### Reversal and instalment posting fixtures

- E7's signed AED −620.00 entry has a linked AED +620.00 inverse, booked Day 6
  and value-dated Day 2. A direct AED 250.00 opening fixture becomes −370.00
  after E7 and 250.00 after E9. In the E1–E7 integration without fees, the
  inverse restores Days 1–6 to 250.00, 250.00, 650.00, 465.00, 465.00, 465.00.
  Full fee/refund balances remain specified in the checkpoint table.
- E9 contributes three records: attempt, outcome, inverse. E10 contributes five:
  attempt, outcome, three instalments. Repeated internal principal correction
  contributes zero; each duplicate input or fresh attempt to reverse an already
  reversed debit contributes three rejection records. Neither adds a posting.
  A fresh AED 0.01 credit after the latter rejection changes the direct fixture
  balance from 250.00 to 250.01, proving processing continues.
- E10's 10,000 BHD minor units allocate as 3,333; 3,333; 3,334, as above.
  Posting fixtures also use two units across three parts (0; 0; 2), zero across
  three parts, and one-part 10.0000 (10,000 units), retaining exact source text.
  BHD 10.0001 rejects excess precision; 9223372036854775.808 is one minor unit
  above the positive endpoint; −0.001 is the smallest representable negative
  amount and rejects sign.
- Opening one unit, a Day 2 debit of one unit, and a Day 3 credit of MaxInt64
  fit before reversal. Restoring the debit fits Day 2 but overflows Day 3 by one
  unit. A stored MinInt64 debit's inverse also overflows checked negation.
- Opening MaxInt64 − 9,999 BHD units plus E10's 10,000 units overflows only at
  its last instalment: the first two parts total 6,666; the final total exceeds
  MaxInt64 by one. No part of the failing batch is appended.

### Closing and fee reconciliation fixtures

- Ordinary closing accepts a supplied positive day; the canonical replay selects
  ordinary closes for Days 1–5 and capitalization-aware finalization for Day 6.
  Day zero and booking day zero are invalid API bounds. Empty correction causes
  reject before append. Alternate-schedule fixtures close Days 6 and 7 ordinarily
  and refund their fees after a one-unit correction, proving that neither the
  canonical capitalization day nor the replay-window end limits the core API.
- Required signed fee net is `−configured charge` for negative eligibility,
  otherwise zero. Append `required net − recorded net`; a zero difference
  appends nothing. Each new close contributes one checkpoint, and each changed
  fee net contributes one posting. No daily accrual record is stored.
- Opening AED −0.01 and a Day 3 credit of 0.01 give Day 2 eligibility −0.01
  and Day 3 eligibility −25.00 after carrying Day 2's fee. Posted closes are
  −25.01 and −50.00. A backdated 0.01 credit restores underlying Day 2 to zero:
  chronological refunds restore posted closes to 0.00 and 0.01. A new 0.01
  debit renews both fees under their original obligations. Two refund/renewal
  cycles produce two assessments, four refunds, and four adjustments; no
  second independent assessment is created. One minor unit is the smallest
  change crossing eligibility; two cycles verify distinct correction identities.
- Negative BHD −0.001 under the explicit zero policy stays −0.001 with zero
  interest and no fee/diagnostic. Custom XYZ at scale four uses a configured
  one-minor-unit charge: −0.0001 closes at −0.0002. These minimal fixtures
  distinguish no-fee configuration from a nonzero policy without built-in codes.
- Two AED accounts, each opening −0.01, receive independent −25.00 fees;
  each account's 0.01 correction/refund restores its own close to zero. Shared
  currency/day/cause values must not share an obligation or correction key.
- Opening MinInt64 cannot absorb the configured 2,500-unit AED fee. Separately,
  opening −1 plus a later-value debit of MinInt64 + 1 equals MinInt64 before
  fees; an earlier 2,500-unit fee would underflow that later close. Both fail
  before appending the fee or close checkpoint.
- Submission interruption fixtures close Days 1–3 at zero, then debit AED 0.01
  value-dated Day 2. Required Days 2/3 fees yield −25.01 and −50.01; Day 1
  stays zero. Failure at append positions one/two/three isolates the input,
  first fee, and second fee batches. A committed input contributes three records;
  each successful fee adds one. Duplicate attempts add only three rejection
  records even while fees remain outstanding.
- Automatic fee-overflow fixtures debit MaxInt64 units, or
  `−(MinInt64 + 2,500)` = 9,223,372,036,854,773,308 units. The former fits as
  principal but cannot absorb its first fee. The latter reaches MinInt64 after
  its first 2,500-unit fee; its second fee underflows. The input and any first
  fee remain committed; the overflowing fee appends nothing.

### Interrupted reconciliation probe

The [P13 test](tests/replay/designlimitation_test.go) uses one AED account at zero,
closes Days 2 and 3, then submits a Day 5 debit of AED 0.01 value-dated Day 2.
Two closed days are the minimum needed to interrupt between fee batches; these
dates reuse the existing submission-interruption fixture. One minor unit is the
smallest overdraft and cannot be halved at AED precision. Each fee is the
task-supplied AED 25.00, not a test-selected tariff.

| Observation | Day 2 close (AED) | Day 3 close (AED) | Fee records |
| --- | ---: | ---: | ---: |
| Before debit | 0.00 | 0.00 | 0 |
| Day 3 fee append interrupted | −25.01 | −25.01 | 1 |
| Expected after failure under whole-submission financial atomicity | 0.00 | 0.00 | 0 |
| Explicit reconciliation completed | −25.01 | −50.01 | 2 |

The tagged assertion's independent oracle is zero: financial atomicity on failure
would preserve the opening zero with no committed debit or fee. The observed
Day 3 balance immediately after the error is −1 − 2,500 = −2,501 minor units,
leaving a 2,500-unit fee outstanding. The completed-work oracle is
−1 − 2,500 − 2,500 = −5,001 minor units; the normal
[interruption regressions](internal/ledger/backdating_test.go) verify that explicit
reconciliation reaches it idempotently. A wrapper interrupts only the Day 3 fee
assessment before storage. The tagged test submits once and queries before any
recovery. All balances are nonpositive, so interest is zero. The atomicity
boundary and trade-off belong to [DESIGN](docs/DESIGN.md#repository-boundary).

## Day 6 boundaries

Use B excluding the final day's own fee and capitalization, S as prior rounded
accruals, and the configured charge C. For B < 0, current interest I = 0 and
F = C if B + S < 0, otherwise zero. For B ≥ 0, F = 0 and I is rounded interest
on B. Capitalization K = S + I; final close = B + K − F.

| B (AED) | S (AED) | I (AED) | F (AED) | K (AED) | Final close (AED) |
| ---: | ---: | ---: | ---: | ---: | ---: |
| −0.01 | 0.02 | 0.00 | 0.00 | 0.02 | 0.01 |
| −0.02 | 0.02 | 0.00 | 0.00 | 0.02 | 0.00 |
| −0.03 | 0.02 | 0.00 | 25.00 | 0.02 | −25.01 |

These are adjacent minor-unit boundaries; halving them changes representability
or boundary coverage. Unit fixtures close Day 5 at AED 37.50 to derive S = 0.02,
then debit 37.51, 37.52, or 37.53 on Day 6 to obtain B. An existing ordinary
Day 6 fee on B = −0.01 must be refunded during finalization under the same
obligation, yielding the first row.

Additional finalization fixtures:

- AED 37.49 on Days 5/6 yields 0.01 each, K = 0.02, and close 37.51.
  The final posted balance would round to 0.02 interest if capitalization leaked
  into its own base; the correct Day 6 accrual remains 0.01.
- With S = 0 and B = BHD −0.001, explicit C = 0 leaves the close −0.001 and
  K = 0.000. Custom XYZ/4 uses the existing one-minor-unit charge fixture:
  B = −0.0001 gives final close −0.0002 and K = 0.0000. A zero credit records
  completion without changing money; one capitalization exists per account/window.
- Opening MaxInt64 cannot absorb its positive daily capitalization; opening
  MinInt64 cannot absorb the AED fee. A Day 6 base of 2,500 minor units earns
  one unit, but a Day 7 credit of MaxInt64 − 2,500 makes capitalization overflow
  that later balance. Each failure leaves the finalization batch uncommitted.
- Maximum-base rounded daily interest is 3,689,348,814,741,910 minor units.
  At 2,500 days its sum is 9,223,372,036,854,775,000; at 2,501 it is
  9,227,061,385,669,516,910, exceeding MaxInt64. Direct close fixtures with
  2,500 or 2,501 prior days isolate overflow when adding current interest or
  accumulating prior interest. These counts straddle the exact sum boundary.

## Expected replay results

| Day | ACC-001 Closing Ledger (AED) | ACC-001 Daily Interest | ACC-002 Closing Ledger (BHD) | ACC-002 Daily Interest |
| :---: | :---: | :---: | :---: | :---: |
| **1** | 250.00 | 0.10 | 0.000 | 0.000 |
| **2** | 250.00 | 0.10 | 0.000 | 0.000 |
| **3** | 650.00 | 0.26 | 0.000 | 0.000 |
| **4** | 465.00 | 0.19 | 0.000 | 0.000 |
| **5** | 465.00 | 0.19 | 10.000 | 0.004 |
| **6** | 466.03 | 0.19 | 10.008 | 0.004 |

> **Note:** ACC-001 Day 6 balance includes AED 1.03 capitalized interest ($\sum \text{Day 1–6 interest}$). ACC-002 Day 6 includes BHD 0.008 capitalized interest.
