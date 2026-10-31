# Acceptance-criterion review

All eight criteria assessed: refuse 2, 7, 8; accept 1 and 6 under documented
human interpretations; accept 3 and 4; accept 5 as a conditional statement
whose premise does not occur for Auth-B. Unspecified extensions and the missing
BHD tariff remain explicit in [AMBIGUITIES.md](AMBIGUITIES.md). Assess each
criterion independently; incorrect ones do not invalidate correct ones.

| # | Verdict | Reason |
| --- | --- | --- |
| 1 | Correct under human-agreed fee-free interpretation | Input prefix E1–E8 excluding all fee entries gives Day 2: 1,200.00 − 950.00 − 620.00 = AED −370.00. This is not actual posted Day 2 as known at Day 5 closing, AED −395.00. Final restatement after E9 gives AED 250.00. TASK.md is unchanged. |
| 2 | Incorrect; refuse | E7 triggers Day 2 and Day 4 fees before E8; Day 5 incurs its own fee at closing. Three AED 25.00 assessments total AED 75.00. Corresponding post-fee closes: AED −395.00, −205.00, −230.00. Day 3 closes AED 5.00 with no fee. E9 refunds all three without erasing their assessment history. Exactly one Day 2 fee violates daily assessment for other negative days. |
| 3 | Correct; final settlement confirmed by human | Accept E5, debit AED 185.00, release Auth-A's full AED 200.00 hold, mark settled. No partial indicator/further captures exist. Unused AED 15.00 becomes available without credit. Later rebuilds do not revoke past authorization decisions. |
| 4 | Correct; confirmed by human | Explicit rejection conflicts with no non-negotiable rule. Reject E6 without debit because Auth-Z is absent; no force-post exception exists. The earlier draft's unresolved verdict was an error, corrected after human review. |
| 5 | Correct conditional statement | Approved holds affect available balance only. Auth-B actually declines at E8 and remains declined after E9, so creates no hold in this replay. Test the approved-hold invariant separately. |
| 6 | Correct under human-agreed financial-restoration interpretation | After E9/corrections and before capitalization, restore historical financial balances, refund unjustified fees, rebuild interest. Linked refunds book Day 6 with original values Days 2, 4, 5; preserve E7/assessments. Corrections idempotent; past authorizations unchanged. This interprets rather than rewrites TASK.md. |
| 7 | Incorrect; refuse | Three BHD 3.334 amounts total 10.002, exceeding E10 by 0.002. Human-agreed split 3.333, 3.333, 3.334 puts residual last and conserves 10.000. |
| 8 | Incorrect; refuse | Discarding a remainder directly violates exact reconciliation of rounded daily accruals to capitalization. Capitalize the corrected rounded daily sum. |

## Abandoned build approaches and discarded specification proposals

No build approach was abandoned: implementation has not begun. Do not invent
build attempts from policy discussions.

The proposal to accept E6 as an unmatched external payment was withdrawn after
human correction: it lacked support in the task and displaced consistent
criterion 4. Replaced by rejection without debit. This was a discarded
specification proposal, not a built implementation.

The identical-payload retry proposal was also declined. The human chose all
duplicate input IDs to reject for a small, predictable implementation; retries
are outside the assignment's stated requirements. Internal correction/closing
idempotence remains required. No retry implementation was built or abandoned.
