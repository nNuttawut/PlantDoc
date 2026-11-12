const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export interface AskResult {
  id: string
  species_name: string
  english_name: string
  scientific_name: string
  answer: string
  keywords: string[]
  image_url: string | null
  cached: boolean
  verified: boolean
}

export async function ask(question: string, image: File | null): Promise<AskResult> {
  const form = new FormData()
  form.append('question', question)
  if (image) form.append('image', image)

  const res = await fetch(`${API_URL}/api/ask`, { method: 'POST', body: form })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error ?? `เกิดข้อผิดพลาด (${res.status})`)
  return data as AskResult
}

// Wake the backend early (free hosting tiers sleep when idle; first request can take ~1 min).
export function wakeBackend() {
  fetch(`${API_URL}/api/health`).catch(() => {})
}
