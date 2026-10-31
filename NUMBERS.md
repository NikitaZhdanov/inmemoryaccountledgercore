# Numerical inventory and expected results

This file owns numerical sources, justifications, calculations, and verification
parameters. [SPEC](docs/SPEC.md) defines behavior; [AMBIGUITIES](AMBIGUITIES.md)
explains business choices. The [Go design](docs/DESIGN.md#language-and-tooling)
uses checked `int64` minor units with the representation bounds below. The
toolchain baseline remains to be recorded when implementation begins.
Calculations and verification use exact equality, without numerical tolerance.

## Constants and sources

| Value | Meaning, units, source | Why this value rather than half |
| --- | --- | --- |
| Days 1–6; six days | TASK.md reporting/accrual window | Half changes the assignment; day labels are discrete. |
| ACC-001/ACC-002; two accounts | TASK.md AED/BHD accounts | Identifiers cannot be halved; omitting an account changes scope. |
| E1–E10; ten events | TASK.md canonical replay sequence | Identifiers cannot be halved; fewer events omit required inputs. |
| AED 0.00; BHD 0.000 | TASK.md opening balances | Zero is its own half; nonzero changes the source. |
| AED 25.00 | TASK.md daily fee | Half violates the specified charge. BHD tariff/FX remain unspecified; see [A5](AMBIGUITIES.md#a5--fee-bases-and-missing-bhd-tariff). |
| Once per account/assessed day | TASK.md assessment frequency | A half-occurrence is meaningless; identity behavior is in [SPEC](docs/SPEC.md#fees). |
| 0.04% = 0.0004 = 1/2,500 per day | TASK.md exact positive-balance interest rate | Half changes the contractual rate. |
| AED two places; BHD three | TASK.md storage/rounding scales | Discrete currency conventions, not adjustable magnitudes. |
| AED 0.01; BHD 0.001 | Minor units, derived as 10^−scale | Half is not representable at the required scale. |
| 100 = 10^2; 1,000 = 10^3 | Derived AED/BHD minor-unit factors | Half changes the unit and invalidates representability checking. |
| Zero | TASK.md inclusive authorization boundary and nonpositive-interest threshold | Zero is its own half. |
| Three instalments | TASK.md E10 count | A fractional count is meaningless; changing the count alters the event. |
| HALF_EVEN | Human-selected daily rounding convention; [A6](AMBIGUITIES.md#a6--interest-rounding-representation-and-finalization) | Discrete mode; no meaningful half. Exact midpoint is half a minor unit by definition. |

### int64 representation

This is a technical choice following the human's preference, not a task-mandated
business limit. Signed 64-bit minor units range from −2^63 to 2^63 − 1 inclusive
(−9,223,372,036,854,775,808 through 9,223,372,036,854,775,807 minor units).
The width and endpoints follow Go's `int64` type; the exponent accounts for the
sign bit. Halving the width to 32 bits unnecessarily reduces headroom; halving
the endpoints introduces an arbitrary cap unrelated to the selected type.
Major-unit bounds are these endpoints divided by the currency's minor-unit
factor. No additional monetary cap or tolerance is selected.

Input parsing/scaling and every intermediate sum, difference, or negation must
fit this range. Reject out-of-range inputs; derived overflow is a technical
failure, not an authorization or fee-policy decision. Do not clamp or wrap.
Boundary verification uses the exact endpoints, one minor unit outside each,
addition/subtraction crossing a bound, negating the minimum, and formatting both
endpoints. These are derived adjacent-boundary cases; half no longer tests the
limit. Maximum-positive-base interest must also calculate safely and exactly.
Divide minor units by the existing rate denominator and round the quotient from
its remainder, without first multiplying the balance by a rate numerator/scale.

### Event inventory

The canonical source is [TASK.md](docs/TASK.md#event-stream-replayed-in-this-order).
Every amount/date below is task-mandated; halving changes the input. Account,
event, and authorization references are identifiers, not quantities. E9's
inverse amount is derived from E7 rather than independently selected.

| Event | Booked day | Value day | Account | Amount/reference |
| --- | --- | --- | --- | --- |
| E1 | 1 | 1 | ACC-001 | Credit AED 1,200.00 |
| E2 | 1 | 1 | ACC-001 | Debit AED 950.00 |
| E3 | 2 | 2 | ACC-001 | Auth-A hold AED 200.00 |
| E4 | 3 | 3 | ACC-001 | Credit AED 400.00 |
| E5 | 4 | 4 | ACC-001 | Auth-A settlement AED 185.00 |
| E6 | 4 | 4 | ACC-001 | Auth-Z settlement AED 180.00 |
| E7 | 5 | 2 | ACC-001 | Debit AED 620.00 |
| E8 | 5 | 5 | ACC-001 | Auth-B hold AED 90.00 |
| E9 | 6 | 2 | ACC-001 | Reverse E7; derived inverse credit AED 620.00 |
| E10 | 5 | 5 | ACC-002 | Credit BHD 10.000 in three instalments |

All results below are independently derived expectations, not adjustable inputs
or executed ledger tests. Halving a derived result would contradict its formula,
the supplied events, or the selected precision. Human choices determine behavior
rather than arbitrary expected balances.

## Checkpoint calculations

Amounts in this section are AED.

- After E2: 1,200.00 − 950.00 = 250.00; E3 availability = 250.00 − 200.00
  = 50.00.
- Before E5: ledger 250.00 + 400.00 = 650.00; available 650.00 − 200.00
  = 450.00. Afterwards: ledger/available 650.00 − 185.00 = 465.00; hold
  zero. Released unused reservation = 200.00 − 185.00 = 15.00.

| Day 2 observation | Calculation | Balance |
| --- | --- | --- |
| Fee-free E1–E8 calculation | 250.00 − 620.00 | −370.00 |
| Posted as known at Day 5 closing | −370.00 − 25.00 | −395.00 |
| Final restatement | −395.00 + 620.00 + 25.00 | 250.00 |

The sequence below covers E7's closed-day reconciliation, E8, and Day 5 closing
before E9; timing is defined in [SPEC](docs/SPEC.md#history-and-replay).

| Assessed day | Pre-own-fee balance | Fee | Post-fee close | Rounded interest |
| --- | --- | --- | --- | --- |
| 2 | −370.00 | 25.00 | −395.00 | 0.00 |
| 3 | 5.00 | 0.00 | 5.00 | 0.00 |
| 4 | −180.00 | 25.00 | −205.00 | 0.00 |
| 5 | −205.00 | 25.00 | −230.00 | 0.00 |

Day 3 pre-own-fee balance = −395.00 + 400.00 = 5.00; interest =
5.00 / 2,500 = 0.002, rounded to 0.00. Day 4 pre-own-fee balance =
5.00 − 185.00 = −180.00. E8's proposed availability = −205.00 − 90.00
= −295.00. Three charges total 3 × 25.00 = 75.00; E9 restoration before
capitalization = −230.00 + 620.00 + 75.00 = 465.00.

Corrected Days 1–5 accruals sum to 0.84; Days 2–5 sum to 0.74, previously
zero after E7/Day 5 closing. These are derived accrual totals, not additional
credits; representation is defined in [SPEC](docs/SPEC.md#interest-and-financial-restoration).

## Instalment conservation

10,000 BHD minor units divided by three gives quotient 3,333 and residual one.
The [selected allocation](docs/SPEC.md#instalments) therefore yields BHD 3.333,
3.333, 3.334, totaling 10.000. Position is discrete; halving amounts breaks
conservation. Three amounts of 3.334 instead total 10.002, excess 0.002.

## Expected replay results

Monetary balances **restated after E1–E10, all corrections, and Day 6
finalization**. ACC-001 is AED; ACC-002 is BHD. Recorded authorization states
and errors follow [SPEC's reporting rules](docs/SPEC.md#reporting-and-observations).

| Day | ACC-001 closing ledger | ACC-001 daily interest | ACC-002 closing ledger | ACC-002 daily interest |
| --- | --- | --- | --- | --- |
| 1 | 250.00 | 0.10 | 0.000 | 0.000 |
| 2 | 250.00 | 0.10 | 0.000 | 0.000 |
| 3 | 650.00 | 0.26 | 0.000 | 0.000 |
| 4 | 465.00 | 0.19 | 0.000 | 0.000 |
| 5 | 465.00 | 0.19 | 10.000 | 0.004 |
| 6 | 466.03 | 0.19 | 10.008 | 0.004 |

For each restored AED 465.00 base, exact daily interest = 0.186, rounded to
0.19. AED capitalization = 0.10 + 0.10 + 0.26 + 0.19 + 0.19 + 0.19 =
1.03; final ledger = 465.00 + 1.03 = 466.03. BHD Day 5 and Day 6 each
earn 10.000 / 2,500 = 0.004; capitalization = 0.008; final ledger = 10.008.
Day 6 interest bases exclude the credits, while eligibility bases include them.

Net fees are zero in every final row. Expected AED assessment/refund records:

| Assessed/value day | Assessment | Assessment booking day | Linked refund | Refund booking day |
| --- | --- | --- | --- | --- |
| 2 | AED 25.00 | 5 | AED 25.00 | 6 |
| 4 | AED 25.00 | 5 | AED 25.00 | 6 |
| 5 | AED 25.00 | 5 | AED 25.00 | 6 |

Charges and refunds each total AED 75.00. No BHD fee or policy error occurs
in this stream. Dates are discrete task/correction dates, not quantities to halve.

## Day 6 boundaries

Apply the [Day 6 equations](docs/SPEC.md#day-6-finalization). No iterative solver
is needed: prior rounded accruals are nonnegative. A negative B makes current
interest zero with or without a fee, leaving capitalization equal to prior
accruals; nonnegative B prevents a fee, so interest uses B directly.

Chosen AED verification cases use prior accrual S = 0.02 and current-day
interest zero:

| B before own fee/capitalization | Capitalization | Fee | Final close |
| --- | --- | --- | --- |
| −0.01 | 0.02 | 0.00 | 0.01 |
| −0.02 | 0.02 | 0.00 | 0.00 |
| −0.03 | 0.02 | 25.00 | −25.01 |

Two minor units of credit allow all starting balances to be negative while
spanning one unit above, exactly at, and one below zero afterwards. Halving
loses that adjacent representable set. Results derive from B + K − F.

## Verification parameters

These examples are chosen verification inputs, not changes to the replay stream.

| Currency | Positive base | Exact daily interest | HALF_EVEN result |
| --- | --- | --- | --- |
| AED | 12.50 | 0.005 | 0.00 |
| AED | 37.50 | 0.015 | 0.02 |
| BHD | 1.250 | 0.0005 | 0.000 |
| BHD | 3.750 | 0.0015 | 0.002 |

Bases deliberately exercise midpoint ties in both directions/currencies; half
a base removes its exact tie. Results derive from the daily-rate formula.

| Principal input | Exact validation result |
| --- | --- |
| AED 1.000 | Accept as equal value AED 1.00 |
| AED 1.005 | Reject |
| BHD 1.0000 | Accept as equal value BHD 1.000 |
| BHD 1.0001 | Reject |

The first input is the human's example; half is also representable but not the
supplied example. The rejected AED midpoint shows why input validation must not
use interest rounding; half would no longer demonstrate that midpoint. The BHD
pair isolates a zero/nonzero fourth digit; halving adds no useful coverage.

Duplicate verification uses two submissions, the minimum to trigger duplication;
half gives one and never exercises rejection. Cover identical/changed payloads,
monetary/authorization inputs, and originally rejected IDs. This count is not a
retry limit. Monetary assertions use exact equality, not tolerances.
