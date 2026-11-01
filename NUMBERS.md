# Numerical inventory and expected results

This file owns numerical sources, justifications, calculations, and verification
parameters. [SPEC](docs/SPEC.md) defines behavior; [AMBIGUITIES](AMBIGUITIES.md)
explains business choices. The [Go design](docs/DESIGN.md#language-and-tooling)
uses checked `int64` minor units with the representation bounds below.
P01 uses Go module `inmemoryaccountledgercore`, baseline Go 1.23.1, observed
toolchain `go1.23.1 darwin/amd64`, and no external dependencies. The module path
is a local import identity, not a network location; the runtime baseline matches
the available toolchain rather than speculating about an untested older version.
Version labels and platform names cannot meaningfully be halved.
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

### P01 money verification

P01 uses literal minor-unit/decimal expectations independent of its parser and
formatter. Test names and array positions are identifiers, not numeric policy.
The decimal radix is 10 (`strconv` base and fixed-point grammar); halving would
interpret different numbers. Scale, zero, unit factors, and `int64` bounds are
already sourced above. Padding uses the scale, and the optional sign and decimal
point each occupy one ASCII character; these are syntax lengths, not tunable limits.

| Fixtures / constants | Meaning, source or derivation | Why these rather than half |
| --- | --- | --- |
| AED 0 / −0.000 → 0 minor units → 0.00; BHD +0.0000 → 0 → 0.000 | Zero/sign normalization and redundant-zero handling | Zero is its own half; syntax variants probe normalization. |
| AED 1 / 1.0 / 1.00 / 1.000 → 100; BHD 1 / 1.0 / 1.00 / 1.000 / 1.0000 → 1,000 | Same principal at different input precisions; human's trailing-zero examples | Halving adds no new representability or padding case. |
| AED 1.2 → 120; BHD 1.2 → 1,200 and 1.23 → 1,230 | Nonzero shorter fractions padded on the right to the currency scale | These distinguish padding direction; halving adds no new path. |
| AED 0.01 → 1; BHD 0.001 → 1; corresponding negatives | Smallest supported currency units | Half is unrepresentable. |
| AED +0012.30 → 1,230; −12.30 → −1,230; BHD 12.345 → 12,345; −00012.345000 → −12,345 | Fixed-point leading zeros, sign, mixed fractional digits, and zero trimming; derived by scale factors | Selected digits exercise both scales and leading fractional/padding behavior; halving adds no new path. |
| AED 1.005 / 0.0001; BHD 1.0001 / −0.0001 | Nonzero excess precision (including a nonzero digit after excess zeros) | Halving is still invalid but loses the AED midpoint example or changes the selected precision probe. |
| AED endpoints 92233720368547758.07 / −92233720368547758.08; BHD 9223372036854775.807 / −9223372036854775.808 | Exact signed bounds divided by the respective factors | Half no longer probes the bounds; appended zeros test exact representability without narrowing range. |
| AED 92233720368547758.08 / −92233720368547758.09; BHD 9223372036854775.808 / −9223372036854775.809 | One minor unit beyond either endpoint | Adjacent boundary probes, not arbitrary amounts to halve. |
| AED whole text 92233720368547759; BHD whole text 9223372036854776; 9223372036854775808 whole text | Scaling overflow and whole-part overflow | Derived from the endpoint's whole part plus one, or maximum integer plus one; half fits and misses overflow. |
| `Max`, `Min`, `Max−1`, `Min+1`, −1, 0, +1 minor units | Checked add/subtract/negate boundary tables | Endpoints and adjacent units expose wrapping, including subtracting `Min` without negating it. Half does not hit these boundaries. |
| Sums `[Max, +1, −1]`, `[Min, −1, +1]` | Intermediate overflow despite representable mathematical total | Minimum cancellation sequence; halving avoids overflow. |
| Sums `[Max, −1, +1]`, `[Min, +1, −1]`, and `[Max, Min] → −1` | Safe ordered accumulation and cancellation | Boundary complements verify no false overflow; half changes endpoint/parity properties. |
| Empty sum → 0 in the requested currency | Additive identity; no count limit | Zero terms cannot be halved. |
| `USD`, empty currency | Unsupported identifier / missing Go zero-value currency | Identifiers have no half; neither implies a conversion or default currency. |
| Empty text, sign-only text, `.1`, `1.`, `1.2.3`, `1e2`, `1,000.00`, `1_000`, whitespace, double/mixed signs, `NaN`, `Inf`, Unicode digits/minus | Lexical rejection fixtures for the documented ASCII fixed-point grammar | Syntax examples, not monetary magnitudes; halving is meaningless. |

Every error-path assertion also checks that no partial money value is returned.
Error identities, rather than message text, distinguish invalid input from derived
overflow. No random seed, fuzz budget, numerical tolerance, or balance cap is chosen.

### Development automation

These are technical choices for the human's lightweight automation request, not
ledger policy or changes to P01's baseline.

| Value | Meaning, source or rationale | Why this value rather than half |
| --- | --- | --- |
| Staticcheck v0.8.2 (2026.2.2); govulncheck v1.9.0 | Verified releases from the official Go module proxy; pinned in Makefile for consistent local/CI analysis | Version identities cannot be halved; explicit pins avoid silently changing analyzers. |
| Go 1.26 minimum for the pinned analyzers; CI `stable` | Analyzer module requirements; CI selects the latest stable Go to check its standard library as well as application code | Version labels cannot be halved. The module baseline remains 1.23.1, not a security guarantee for that old compiler. |
| Go 1.27.3 for local automation verification | Observed stable release from Go's official download index; selected with `GOTOOLCHAIN=go1.27.3` for local `make ci` and `make check` | Version identity cannot be halved; this records the actual checked compiler, not a new module baseline. |
| One Linux job, `ubuntu-latest` | Small CI footprint using a GitHub-hosted runner with race-detector support | Half a job is meaningless; an OS/version matrix adds work without a current platform-specific requirement. |
| 10 minutes | CI job timeout, allowing cold compilation of both analyzers and race instrumentation while bounding stalled runs | Five minutes leaves less cold-cache margin; this is an operational ceiling, not a performance target. |
| `23 6 * * 1` | Weekly vulnerability refresh, Monday 06:23 UTC, away from the top-of-hour schedule peak | Calendar fields are discrete; the exact offset has no business meaning. Halving the weekly interval doubles unattended work. |
| Weekly Dependabot interval; schema version 2 | Low-volume updates to pinned GitHub Actions only | Half the interval adds update churn; schema version is an API identity, not a magnitude. |
| checkout v7.0.1; setup-go v7.0.0; their release commit SHAs | Official action release identities, pinned in CI | Release labels/SHAs cannot be halved; hashes identify reviewed code. |
| Test `-count=1` | Race/coverage run bypasses cached test results | A half execution is meaningless; a single fresh run gives the requested evidence without repetition. |
| `coverage.out`; no coverage threshold | Go coverage profile for local inspection; excluded by the existing ignore rule | Artifact name is an identifier; no arbitrary percentage is selected to halve. |
| Shell exit status 1 for formatting violations | Conventional unsuccessful Make target, also used by the formatter rejection probe | Status is discrete; halving is meaningless. |
