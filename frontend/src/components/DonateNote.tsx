// Shown inside the loading box when an answer takes longer than 10 seconds.
// The QR image lives in public/donate-qr.jpg (Krungthai PromptPay QR).
const QR_SRC = '/donate-qr.jpg'
const ACCOUNT_NAME = 'Nuttawut Likitwongjinda'

export default function DonateNote() {
  return (
    <div className="donate">
      <p className="donate-title">⏳ วันนี้ตอบช้ากว่าปกติเล็กน้อย</p>
      <p>
        พืชสวนโรคใช้ AI รุ่นฟรี ซึ่งมีคนใช้พร้อมกันเยอะ <strong>คำตอบกำลังมา ไม่ต้องปิดหน้านี้นะครับ</strong>
      </p>
      <p>
        💚 ถ้าอยากให้พืชสวนโรคตอบได้เร็วขึ้น ร่วมสนับสนุนค่า AI ได้ตามกำลัง
        ทุกบาทช่วยให้เกษตรกรได้ใช้ฟรีต่อไป
      </p>

      <div className="donate-qr">
        <img src={QR_SRC} alt={`QR พร้อมเพย์ ธนาคารกรุงไทย ชื่อบัญชี ${ACCOUNT_NAME}`} loading="lazy" />
        <a className="btn btn-outline" href={QR_SRC} download="พร้อมเพย์-พืชสวนโรค.jpg">
          💾 บันทึกรูป QR ไปสแกนในแอปธนาคาร
        </a>
        <p className="donate-label">พร้อมเพย์ · กรุงไทย · {ACCOUNT_NAME}</p>
      </div>
    </div>
  )
}
