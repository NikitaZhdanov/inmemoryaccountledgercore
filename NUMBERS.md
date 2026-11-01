# Numerical Inventory

This document records all numerical sources, explicit parameters, derived values, and precision bounds used in the ledger core.

## 1. Domain Constants

| Parameter | Value | Source / Business Rationale |
| :--- | :--- | :--- |
| **Replay Window** | 6 days (Day 1 – Day 6) | Sequence bounds defined by task requirements. |
| **Accounts** | `ACC-001` (AED), `ACC-002` (BHD) | Opening balance for both is `0.00` / `0.000`. |
| **AED Overdraft Fee** | AED 25.00 | Assessed once per day on negative closing ledger balances. |
| **BHD Overdraft Fee** | BHD 0.000 | Explicit zero-fee rule (no fee tariff specified for BHD). |
| **Daily Interest Rate** | 0.04% ($0.0004$) per day | Accrued on positive closing ledger balances; capitalized on Day 6. |

---

## 2. Precision & Representation

| Currency | Scale (Decimals) | Factor | Storage Format | Min / Max Bounds |
| :--- | :--- | :--- | :--- | :--- |
| **AED** | 2 | $10^2 = 100$ | Signed `int64` minor units | $\pm 92,233,720,368,547,758.07$ AED |
| **BHD** | 3 | $10^3 = 1,000$ | Signed `int64` minor units | $\pm 9,223,372,036,854,775.807$ BHD |

- **Arithmetic Safety:** All calculations use checked `int64` arithmetic. Operations that overflow or underflow the minor unit bounds are rejected with errors.
- **Input Validation:** Amounts matching exact scale are accepted. Inputs with excess decimal precision (e.g., AED `1.005`) are strictly rejected.

---

## 3. Allocation & Rounding Rules

- **Interest Accrual Rounding:** Calculated daily using `HALF_EVEN` (Banker's Rounding) to the currency's decimal precision.
- **Instalment Allocation (E10):** Splitting BHD 10.000 across 3 equal payments produces quotient BHD 3.333 and a 1-minor-unit residual. Per policy, remainder is added to the **last instalment**:
  - Instalment 1: BHD 3.333
  - Instalment 2: BHD 3.333
  - Instalment 3: BHD 3.334 (Sum = BHD 10.000)

---

## 4. Expected Replay Balances

| Day | ACC-001 Closing Ledger (AED) | ACC-001 Daily Interest | ACC-002 Closing Ledger (BHD) | ACC-002 Daily Interest |
| :---: | :---: | :---: | :---: | :---: |
| **1** | 250.00 | 0.10 | 0.000 | 0.000 |
| **2** | 250.00 | 0.10 | 0.000 | 0.000 |
| **3** | 650.00 | 0.26 | 0.000 | 0.000 |
| **4** | 465.00 | 0.19 | 0.000 | 0.000 |
| **5** | 465.00 | 0.19 | 10.000 | 0.004 |
| **6** | 466.03 | 0.19 | 10.008 | 0.004 |

> **Note:** ACC-001 Day 6 balance includes AED 1.03 capitalized interest ($\sum \text{Day 1–6 interest}$). ACC-002 Day 6 includes BHD 0.008 capitalized interest.
