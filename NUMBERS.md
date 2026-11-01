# Numerical inventory

Task constants, selected numerical conventions, and independently derived replay
results. [TASK](docs/TASK.md) owns the supplied event amounts, dates, and order;
[AMBIGUITIES](AMBIGUITIES.md) owns interpretation rationale.

## Constants and sources

| Parameter | Value | Source or reason |
| --- | --- | --- |
| Replay window | Days 1–6; capitalize at end of Day 6 | Supplied by TASK. |
| Opening balances | ACC-001 AED 0.00; ACC-002 BHD 0.000 | Supplied by TASK. |
| Overdraft fee | AED 25.00, once per account per negative closing day | Supplied by TASK. |
| Daily interest rate | 0.04% = 0.0004 = 1/2,500 | Supplied by TASK; applies to positive closing balances. |
| Authorization approval boundary | Available balance after the hold ≥ 0 | Supplied by TASK. |
| E10 instalment count | 3 | Supplied by TASK. |
| BHD fee configuration | BHD 0.000 | Explicit assumption: no BHD tariff or exchange rate is supplied. BHD never becomes negative in this replay. See [A4](AMBIGUITIES.md#a4--fee-bases-and-missing-bhd-tariff). |

The supplied amounts, dates, and counts are fixed inputs; halving them changes the
assignment. The rate denominator 2,500 is derived, not chosen: halving it doubles
the interest rate. Zero is the specified opening/approval boundary and the
explicit BHD no-fee configuration.

## Currency precision

| Currency | Decimal places | Minor units per unit | Smallest amount |
| --- | ---: | ---: | ---: |
| AED | 2 | 100 | 0.01 |
| BHD | 3 | 1,000 | 0.001 |

The scales are supplied; the factors are 10 raised to those scales. Half a minor
unit cannot be represented at the required precision. Principal must be exactly
representable; calculated interest uses the rounding convention below.

## Interest calculation

For each day, round max(closing interest base, 0) × 0.0004 to the account's
precision using HALF_EVEN. At an exact midpoint, select the even minor-unit
count. This convention reduces midpoint tie bias; TASK does not specify a mode.
See [A5](AMBIGUITIES.md#a5--interest-rounding-representation-and-finalization).

In minor units, divide a positive base by 2,500. Round the quotient upward if the
remainder exceeds 1,250, or equals 1,250 and the quotient is odd. The midpoint
1,250 is half the rate denominator. Nonpositive bases accrue zero.

Capitalize the exact sum of the six corrected, rounded daily accruals. The Day 6
credit earns no same-day interest. Its amount is included in final fee eligibility;
[SPEC](docs/SPEC.md#day-6-finalization) owns the calculation order.

## Instalment conservation

E10 supplies 10,000 BHD minor units across three instalments:
10,000 = 3 × 3,333 + 1. Allocate BHD 3.333, 3.333, and 3.334, totaling 10.000.
Putting the residual in the last instalment is the selected balancing convention
([A8](AMBIGUITIES.md#a8--currency-precision-and-instalments)). The residual is the
smallest BHD unit and cannot be halved. Three credits of 3.334 would total 10.002.

## Checkpoint calculations

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

E7 causes three AED 25.00 assessments, for Days 2, 4, and 5: total AED 75.00.
Earlier fees carry into later balances. Day 4 eligibility before its own fee is
−180.00; Day 5 eligibility is −205.00. E9 appends AED 620.00 and three linked
AED 25.00 refunds, restoring net fees to zero without erasing assessments.

Auth-A's AED 200.00 hold leaves availability 250.00 − 200.00 = 50.00 on Day 2
and 650.00 − 200.00 = 450.00 on Day 3. E5 releases the entire hold and debits
185.00, leaving ledger and availability at 465.00. E6's unmatched 180.00
settlement is rejected without a debit.

At E8, before Day 5's fee, availability is −205.00; the requested 90.00 would
produce −295.00. Auth-B declines and creates no hold; E9 preserves that decision.

After E7 and Day 5 close, AED daily accruals are 0.10, 0.00, 0.00, 0.00, 0.00.
Day 3's base 5.00 gives unrounded interest 0.002, rounding to 0.00. E9 restores
the accruals shown below. E10 affects only BHD and retains value Day 5, earning
0.004 on both Days 5 and 6 despite arriving after E9.

## Expected replay results

Balances are restated after E1–E10 and Day 6 finalization.

| Day | ACC-001 closing ledger (AED) | Daily interest (AED) | ACC-002 closing ledger (BHD) | Daily interest (BHD) |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 250.00 | 0.10 | 0.000 | 0.000 |
| 2 | 250.00 | 0.10 | 0.000 | 0.000 |
| 3 | 650.00 | 0.26 | 0.000 | 0.000 |
| 4 | 465.00 | 0.19 | 0.000 | 0.000 |
| 5 | 465.00 | 0.19 | 10.000 | 0.004 |
| 6 | 466.03 | 0.19 | 10.008 | 0.004 |

Single Day 6 credits:

- AED: 0.10 + 0.10 + 0.26 + 0.19 + 0.19 + 0.19 = 1.03; 465.00 + 1.03 = 466.03.
- BHD: 0.004 + 0.004 = 0.008; 10.000 + 0.008 = 10.008.

## Interrupted reconciliation probe

The [required failing test](tests/replay/designlimitation_test.go) uses an AED
0.01 debit, the smallest representable overdraft, and two closed days, the minimum
needed to interrupt between daily fee batches. Each fee is the task's AED 25.00.
Financial atomicity would preserve the prior zero balance after failure; the
observed balance is −0.01 − 25.00 = −25.01. Completing reconciliation yields
−0.01 − 25.00 − 25.00 = −50.01. The chosen smallest debit cannot be halved at AED
precision; one closed day would not expose a failure between fee batches. See
[the design limitation](docs/DESIGN.md#repository-boundary).
