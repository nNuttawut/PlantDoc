import { useEffect, useState } from 'react'
import Markdown from 'react-markdown'
import type { AskResult } from '../api/client'
import { canSpeak, speak, stopSpeaking } from '../lib/speech'

interface Props {
  result: AskResult
  onAskAgain: () => void
}

export default function AnswerCard({ result, onAskAgain }: Props) {
  const [speaking, setSpeaking] = useState(false)
  const identified = result.species_name && result.species_name !== 'ไม่สามารถระบุได้'

  useEffect(() => stopSpeaking, []) // stop reading when leaving this answer

  const toggleSpeak = () => {
    if (speaking) {
      stopSpeaking()
      setSpeaking(false)
      return
    }
    setSpeaking(true)
    const intro = identified ? `ชนิด ${result.species_name}. ` : ''
    speak(intro + result.answer, () => setSpeaking(false))
  }

  return (
    <section className="card answer" aria-live="polite">
      <div className="species-box">
        <div className="species-label">ผลการตรวจ: น่าจะเป็น</div>
        <div className="species-name">{identified ? result.species_name : 'ยังระบุชนิดไม่ได้'}</div>
        {identified && result.scientific_name && <div className="species-sci">{result.scientific_name}</div>}
      </div>

      {result.verified && <p className="note note-verified">✔ คำตอบนี้ผู้เชี่ยวชาญตรวจสอบแล้ว</p>}

      {canSpeak && (
        <button type="button" className="btn btn-big btn-light" onClick={toggleSpeak}>
          <span className="icon" aria-hidden>{speaking ? '⏹️' : '🔊'}</span>
          {speaking ? 'หยุดอ่าน' : 'ฟังคำตอบ'}
        </button>
      )}

      {result.image_url && <img className="answer-img" src={result.image_url} alt="รูปที่ส่งมา" />}

      <div className="answer-body">
        <Markdown>{result.answer}</Markdown>
      </div>

      <p className="note note-warn">
        ⚠️ เป็นคำแนะนำเบื้องต้นเท่านั้น ถ้าอาการหนักหรือไม่ดีขึ้น
        ควรปรึกษาเจ้าหน้าที่เกษตรอำเภอ หรือสัตวแพทย์
      </p>

      <button type="button" className="btn btn-big btn-primary" onClick={onAskAgain}>
        🔄 ถามเรื่องใหม่
      </button>
    </section>
  )
}
