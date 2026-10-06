export interface HistoryRecord { kind: 'call' | 'sms'; id: number; from_number: string; to_number: string }
export type HistoryScope = 'local' | 'both'
export interface HistoryResult { record: HistoryRecord; deleted: boolean; error?: string }

// Collect complete threads before asking for deletion confirmation. This only
// reads history; deletion still uses the explicit, per-record confirmation path.
export async function collectConversationHistory(
  numbers: string[], did: number,
  fetchPage: (number: string, did: number, offset: number, limit: number) => Promise<{data: Omit<HistoryRecord, 'kind'>[]; has_more: boolean}>
): Promise<HistoryRecord[]> {
  const records = new Map<number, HistoryRecord>()
  for (const number of new Set(numbers)) {
    let offset = 0
    for (let pageNumber = 0; ; pageNumber++) {
      if (pageNumber >= 10000) throw new Error('Conversation history is too large to prepare in one operation.')
      const page = await fetchPage(number, did, offset, 100)
      for (const message of page.data) records.set(message.id, {...message, kind: 'sms'})
      if (!page.has_more) break
      if (!page.data.length) throw new Error('Conversation history pagination did not advance.')
      offset += page.data.length
    }
  }
  return [...records.values()]
}

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
