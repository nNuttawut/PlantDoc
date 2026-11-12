// Thin wrappers over the Web Speech API (Thai). Each feature is optional:
// components hide the related button when the browser doesn't support it.

type Recognition = {
  lang: string
  interimResults: boolean
  continuous: boolean
  onresult: ((e: { results: ArrayLike<ArrayLike<{ transcript: string }>> }) => void) | null
  onend: (() => void) | null
  onerror: (() => void) | null
  start: () => void
  stop: () => void
}

const RecognitionCtor: (new () => Recognition) | undefined =
  (window as any).SpeechRecognition ?? (window as any).webkitSpeechRecognition

export const canListen = !!RecognitionCtor
export const canSpeak = 'speechSynthesis' in window

export function listen(onText: (text: string) => void, onEnd: () => void): () => void {
  if (!RecognitionCtor) return () => {}
  const rec = new RecognitionCtor()
  rec.lang = 'th-TH'
  rec.interimResults = false
  rec.continuous = false
  rec.onresult = (e) => {
    const text = Array.from(e.results).map((r) => r[0].transcript).join(' ')
    if (text) onText(text)
  }
  rec.onend = onEnd
  rec.onerror = onEnd
  rec.start()
  return () => rec.stop()
}

// Strip Markdown symbols so the voice doesn't read "#" or "*".
function plain(md: string) {
  return md.replace(/[#*_`>|-]+/g, ' ').replace(/\s+/g, ' ').trim()
}

export function speak(markdown: string, onEnd: () => void) {
  if (!canSpeak) return
  window.speechSynthesis.cancel()
  const u = new SpeechSynthesisUtterance(plain(markdown))
  u.lang = 'th-TH'
  u.rate = 0.9 // slightly slower for older listeners
  const thai = window.speechSynthesis.getVoices().find((v) => v.lang.startsWith('th'))
  if (thai) u.voice = thai
  u.onend = onEnd
  u.onerror = onEnd
  window.speechSynthesis.speak(u)
}

export function stopSpeaking() {
  if (canSpeak) window.speechSynthesis.cancel()
}

// Read a short text (e.g. a species name) in the given language.
export function sayText(text: string, lang: 'th-TH' | 'en-US', onEnd?: () => void) {
  if (!canSpeak || !text) return
  window.speechSynthesis.cancel()
  const u = new SpeechSynthesisUtterance(text)
  u.lang = lang
  u.rate = 0.85
  const prefix = lang.slice(0, 2)
  const voice = window.speechSynthesis.getVoices().find((v) => v.lang.replace('_', '-').startsWith(prefix))
  if (voice) u.voice = voice
  u.onend = () => onEnd?.()
  u.onerror = () => onEnd?.()
  window.speechSynthesis.speak(u)
}
