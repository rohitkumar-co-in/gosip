export interface HistoryRecord { kind: 'call' | 'sms'; id: number; from_number: string; to_number: string }
export type HistoryScope = 'local' | 'both'
export interface HistoryResult { record: HistoryRecord; deleted: boolean; error?: string }

// Individual requests retain the server's confirmation, permissions, provider
// checks and audit trail. A failed item never prevents processing later items.
export async function deleteHistoryBatch(
  records: HistoryRecord[], scope: HistoryScope,
  remove: (record: HistoryRecord, scope: HistoryScope) => Promise<unknown>,
  progress: (processed: number) => void = () => {}
): Promise<HistoryResult[]> {
  const results: HistoryResult[] = []
  const seen = new Set<string>()
  for (const record of records) {
    const key = `${record.kind}-${record.id}`
    if (seen.has(key)) continue
    seen.add(key)
    try { await remove(record, scope); results.push({record, deleted: true}) }
    catch (error: unknown) {
      const message = (error as {response?: {data?: {error?: {message?: string}}}})?.response?.data?.error?.message
      results.push({record, deleted: false, error: message || 'Deletion could not be verified. Refresh history before retrying.'})
    }
    progress(results.length)
  }
  return results
}
