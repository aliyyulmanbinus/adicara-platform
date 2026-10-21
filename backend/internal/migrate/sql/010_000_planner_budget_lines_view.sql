-- +goose Up
-- Model baca tunggal untuk semua angka anggaran (matriks, KPI, cashflow, dashboard).
-- Pos dengan rollup_source = 'shopping' mengambil estimasi dan terbayar dari
-- planner_shopping_items; lainnya memakai kolomnya sendiri. Status pelunasan
-- dihitung di sini, bukan disimpan, sehingga tidak bisa menyimpang.
CREATE VIEW planner_budget_lines AS
SELECT x.*,
       x.estimate - x.paid AS remaining,
       CASE
           WHEN x.paid = 0 THEN 'unpaid'
           WHEN x.estimate > 0 AND x.paid >= x.estimate THEN 'settled'
           WHEN x.rollup_source IS NULL AND x.dp_amount > 0 AND x.term_1_amount = 0 AND x.term_2_amount = 0 THEN 'dp_paid'
           ELSE 'partial'
       END AS payment_status
FROM (
    SELECT b.*,
           CASE WHEN b.rollup_source = 'shopping' THEN COALESCE(s.estimate_total, 0) ELSE b.initial_estimate END AS estimate,
           CASE WHEN b.rollup_source = 'shopping' THEN COALESCE(s.actual_total, 0) ELSE b.paid_total END AS paid
    FROM planner_budget_items b
    LEFT JOIN LATERAL (
        SELECT SUM(i.estimated_total)::bigint AS estimate_total,
               SUM(CASE WHEN i.status = 'selesai' THEN i.actual_total ELSE 0 END)::bigint AS actual_total
        FROM planner_shopping_items i
        WHERE i.budget_item_id = b.id
    ) s ON b.rollup_source = 'shopping'
) x;

-- +goose Down
DROP VIEW IF EXISTS planner_budget_lines;
