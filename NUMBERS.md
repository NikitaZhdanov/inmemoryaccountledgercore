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
