import { useEffect, useRef, useState } from 'react'
import { ask, wakeBackend, type AskResult } from './api/client'
import AskForm from './components/AskForm'
import AnswerCard from './components/AnswerCard'

const WAIT_MESSAGES = [
  'กำลังดูรูปและอ่านอาการ…',
  'กำลังตรวจว่าเป็นพืชหรือสัตว์ชนิดไหน…',
  'กำลังหาวิธีรักษาที่เหมาะสม…',
  'ใกล้เสร็จแล้ว รอสักครู่นะครับ…',
]

export default function App() {
  const [loading, setLoading] = useState(false)
  const [waitStep, setWaitStep] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<AskResult | null>(null)
  const [formKey, setFormKey] = useState(0)
  const topRef = useRef<HTMLDivElement>(null)

  useEffect(wakeBackend, [])

  useEffect(() => {
    if (!loading) return
    setWaitStep(0)
    const t = setInterval(() => setWaitStep((s) => Math.min(s + 1, WAIT_MESSAGES.length - 1)), 6000)
    return () => clearInterval(t)
  }, [loading])

  const scrollTop = () => topRef.current?.scrollIntoView({ behavior: 'smooth' })

  const handleAsk = async (question: string, image: File | null) => {
    setLoading(true)
    setError(null)
    setResult(null)
    scrollTop()
    try {
      setResult(await ask(question, image))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'เกิดข้อผิดพลาด')
    } finally {
      setLoading(false)
      scrollTop()
    }
  }

  const askAgain = () => {
    setResult(null)
    setError(null)
    setFormKey((k) => k + 1) // reset the form
    scrollTop()
  }

  return (
    <>
      <header className="site-header">
        <div className="container">
          <h1>🌱 พืชสวนโรค</h1>
          <p>ปรึกษาโรคพืชและสัตว์เลี้ยง ฟรี</p>
        </div>
      </header>

      <main className="container" ref={topRef}>
        {loading && (
          <div className="card loading" role="status">
            <div className="spinner" aria-hidden />
            <p className="loading-text">{WAIT_MESSAGES[waitStep]}</p>
            <p className="hint">อาจใช้เวลาประมาณครึ่งนาที</p>
          </div>
        )}

        {error && !loading && (
          <div className="card error" role="alert">
            <p className="error-title">😟 ขออภัย ตอนนี้ตอบไม่ได้</p>
            <p>{error}</p>
            <p className="hint">ลองกดปุ่ม "ถามพืชสวนโรค" อีกครั้ง</p>
          </div>
        )}

        {result && !loading ? (
          <AnswerCard result={result} onAskAgain={askAgain} />
        ) : (
          <div hidden={loading}>
            <AskForm key={formKey} loading={loading} onSubmit={handleAsk} />
          </div>
        )}
      </main>

      <footer className="site-footer">
        <div className="container">
          <div className="footer-emoji" aria-hidden>🌱 🌾 🌽 🥭 🐄 🐔</div>
          <p>พืชสวนโรค · คำแนะนำโดยปัญญาประดิษฐ์ ใช้ประกอบการตัดสินใจเท่านั้น</p>
          <p className="copyright">© {new Date().getFullYear()} สงวนลิขสิทธิ์ by นัฐวุฒิ ขิตวงศ์จินดา</p>
        </div>
      </footer>
    </>
  )
}
