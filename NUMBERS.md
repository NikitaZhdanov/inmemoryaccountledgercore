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
P05 additionally posts E1/E2/E4 through the core; later event processing remains
planned. All balances below are AED.

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
E9 restores 0.10, 0.10, 0.26, 0.19, 0.19. Late E10 changes BHD Day 5
from 0.000 to 0.004 without changing AED. Final values appear below.

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
