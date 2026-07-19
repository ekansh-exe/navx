// Currency values are integers in the smallest unit (1 currency = 100 units)
// per API_ENDPOINTS.md conventions — never render the raw integer.

export function formatCurrency(units: number): string {
  return (units / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export function formatSignedCurrency(units: number): string {
  const sign = units > 0 ? "+" : units < 0 ? "-" : "";
  return `${sign}${formatCurrency(Math.abs(units))}`;
}

export function formatShares(shares: number): string {
  return shares.toLocaleString();
}

export function formatPercent(value: number, fractionDigits = 2): string {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(fractionDigits)}%`;
}
