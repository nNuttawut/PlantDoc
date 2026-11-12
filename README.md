# PlantDoc
พืชสวนโรค — ระบบถาม-ตอบโรคพืช/สัตว์เลี้ยง ด้วย Gemini + ฐานความรู้ที่สะสมเอง

## Workflow
```
React ──(คำถาม + รูป)──▶ Go /api/ask
  1. Gemini ระบุชนิด + สกัด keyword  (เช่น species="ข้าว", keywords=["ใบเหลือง","เพลี้ยกระโดด"])
  2. ค้นใน Postgres: species ต้องตรงกัน + keywords && $1 (ตรงกันอย่างน้อย 1–3 คำ)
     ├─ เจอ   → ส่งคำตอบเดิม (cached=true) — คำตอบที่ is_verified=true ได้ก่อน
     └─ ไม่เจอ → Gemini ระบุชนิด (ชื่อไทย + ชื่อวิทยาศาสตร์) + วิเคราะห์โรค + วิธีแก้
                → อัปโหลดรูปขึ้น Cloudinary / Supabase Storage
                → บันทึก คำถาม/keyword/URL รูป/คำตอบ ลง DB → ส่งกลับ (cached=false)
```

## Tech stack
| ส่วน | ใช้ |
|---|---|
| Frontend | React + Vite + TypeScript (`frontend/`) |
| Backend | Go 1.22+ net/http, pgx (`backend/`) |
| Database | Supabase PostgreSQL |
| Image storage | Cloudinary (default) หรือ Supabase Storage — เลือกด้วย `STORAGE_PROVIDER` |
| AI | Gemini API (`GEMINI_MODEL`, default `gemini-3.8-flash`) |

## โครงสร้าง
```
backend/
  cmd/server/main.go            entrypoint, wiring, graceful shutdown
  internal/config/              โหลด env
  internal/ai/gemini.go         ExtractKeywords, Diagnose (Gemini REST)
  internal/storage/             Uploader interface + cloudinary.go, supabase.go
  internal/repository/          SQL ตาราง consultations
  internal/service/diagnosis.go โฟลว์หลัก (keyword → cache → AI → save)
  internal/handler/             HTTP routes, CORS, logger
  migrations/001_init.sql       schema
frontend/
  src/api/client.ts             เรียก /api/ask
  src/components/               AskForm, AnswerCard
  src/App.tsx
```

## API
| Method | Path | หมายเหตุ |
|---|---|---|
| GET | `/api/health` | |
| POST | `/api/ask` | multipart: `question` (text), `image` (file, ไม่บังคับ) |
| GET | `/api/consultations?limit=20` | ประวัติการปรึกษาล่าสุด |

## เริ่มใช้งาน (local)
1. **Supabase**: สร้างโปรเจกต์ → SQL Editor → รัน `backend/migrations/001_init.sql` และ `002_add_species.sql` ตามลำดับ
   (`DATABASE_URL` ใช้ **Session pooler** — Direct connection เป็น IPv6 อย่างเดียว)
   (ถ้าใช้ Supabase Storage ให้สร้าง bucket ชื่อ `plantdoc` แบบ **Public**)
2. **Backend**
   ```bash
   cd backend
   cp .env.example .env   # ใส่ DATABASE_URL, GEMINI_API_KEY, Cloudinary keys
   make run               # http://localhost:8080
   ```
3. **Frontend**
   ```bash
   cd frontend
   cp .env.example .env   # VITE_API_URL=http://localhost:8080
   npm install
   npm run dev            # http://localhost:5173
   ```

## หมายเหตุ
- รูปจะถูกส่งให้ Gemini แบบ inline (bytes) ไม่ใช่ URL เพราะ Gemini API ไม่ดึงรูปจาก URL ภายนอกโดยตรง
- อัปโหลดรูปเฉพาะคำถามใหม่ (cache miss) เพื่อประหยัดพื้นที่
- ต่อยอด: คอลัมน์ `question_embedding VECTOR(768)` เตรียมไว้แล้ว — ทำ semantic search ด้วย Gemini embeddings แทน keyword ตรงตัว
- Backend ต่อ DB ด้วย role `postgres` (ผ่าน `DATABASE_URL`) จึงไม่ติด RLS; policy ใน migration มีผลกับการเรียกผ่าน Supabase API (anon key) เท่านั้น
