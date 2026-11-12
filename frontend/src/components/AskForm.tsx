import { useEffect, useRef, useState, type FormEvent } from 'react'
import { shrinkImage } from '../lib/image'
import { canListen, listen } from '../lib/speech'

interface Props {
  loading: boolean
  onSubmit: (question: string, image: File | null) => void
}

export default function AskForm({ loading, onSubmit }: Props) {
  const [question, setQuestion] = useState('')
  const [image, setImage] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [listening, setListening] = useState(false)
  const stopListening = useRef<() => void>(() => {})
  const cameraInput = useRef<HTMLInputElement>(null)
  const galleryInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!image) return setPreview(null)
    const url = URL.createObjectURL(image)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [image])

  const pickImage = async (file: File | undefined) => {
    if (file) setImage(await shrinkImage(file))
  }

  const toggleMic = () => {
    if (listening) return stopListening.current()
    setListening(true)
    stopListening.current = listen(
      (text) => setQuestion((q) => (q ? q + ' ' : '') + text),
      () => setListening(false),
    )
  }

  const canSend = !loading && (question.trim() !== '' || image !== null)

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (canSend) onSubmit(question.trim(), image)
  }

  return (
    <form className="card" onSubmit={handleSubmit}>
      {/* Step 1: photo */}
      <section className="step">
        <h2><span className="step-no">1</span> ถ่ายรูปต้นไม้หรือสัตว์ที่มีอาการ</h2>

        {preview ? (
          <div className="preview">
            <img src={preview} alt="รูปที่เลือก" />
            <button type="button" className="btn btn-outline" onClick={() => setImage(null)} disabled={loading}>
              ✕ เอารูปออก
            </button>
          </div>
        ) : (
          <div className="photo-buttons">
            <button type="button" className="btn btn-big" onClick={() => cameraInput.current?.click()} disabled={loading}>
              <span className="icon" aria-hidden>📷</span> ถ่ายรูป
            </button>
            <button type="button" className="btn btn-big btn-light" onClick={() => galleryInput.current?.click()} disabled={loading}>
              <span className="icon" aria-hidden>🖼️</span> เลือกรูปในเครื่อง
            </button>
          </div>
        )}
        <p className="hint">ถ่ายให้เห็นใบ ลำต้น หรือจุดที่มีอาการชัดๆ (ไม่มีรูปก็ได้)</p>

        <input ref={cameraInput} type="file" accept="image/*" capture="environment" hidden
          onChange={(e) => { pickImage(e.target.files?.[0]); e.target.value = '' }} />
        <input ref={galleryInput} type="file" accept="image/*" hidden
          onChange={(e) => { pickImage(e.target.files?.[0]); e.target.value = '' }} />
      </section>

      {/* Step 2: describe */}
      <section className="step">
        <h2><span className="step-no">2</span> เล่าอาการให้พืชสวนโรคฟัง</h2>
        <label htmlFor="question" className="sr-only">อาการ</label>
        <textarea
          id="question"
          rows={4}
          placeholder="เช่น ใบข้าวเหลือง มีแมลงสีน้ำตาลเกาะโคนต้น"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          disabled={loading}
        />
        {canListen && (
          <button type="button" className={`btn btn-outline btn-mic ${listening ? 'is-on' : ''}`}
            onClick={toggleMic} disabled={loading} aria-pressed={listening}>
            <span className="icon" aria-hidden>🎤</span>
            {listening ? 'กำลังฟัง… พูดได้เลย (กดอีกครั้งเพื่อหยุด)' : 'กดแล้วพูดแทนการพิมพ์'}
          </button>
        )}
      </section>

      {/* Step 3: send */}
      <section className="step">
        <h2><span className="step-no">3</span> กดปุ่มเพื่อถามพืชสวนโรค</h2>
        <button type="submit" className="btn btn-big btn-primary" disabled={!canSend}>
          {loading ? 'กำลังตรวจอาการ…' : '🔍 ถามพืชสวนโรค'}
        </button>
        {!canSend && !loading && <p className="hint">ถ่ายรูป หรือ เล่าอาการ อย่างใดอย่างหนึ่งก่อนนะครับ</p>}
      </section>
    </form>
  )
}
