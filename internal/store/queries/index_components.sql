-- name: ListIndexComponentPrices :many
SELECT ic.weight, c.current_price
FROM index_components ic
JOIN cards c ON c.id = ic.component_card_id
WHERE ic.index_card_id = $1;
