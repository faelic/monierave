export function createIdempotencyKey() {
  return crypto.randomUUID();
}
