-- 1. เปิดใช้ Extension สำหรับทำ Vector Search (เผื่ออนาคตหากต้องการค้นหาด้วยความหมาย)
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. สร้างตารางหลักสำหรับเก็บประวัติการปรึกษาและคำตอบ
CREATE TABLE public.consultations (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL, -- ผูกกับระบบ Login Supabase (ปล่อย Null ได้ถ้าทำเป็น Guest)
    question_text TEXT NOT NULL,             -- คำถามที่เกษตรกรพิมพ์
    image_url TEXT,                          -- URL รูปภาพ (จาก Cloudinary หรือ Supabase Storage)
    keywords TEXT[] DEFAULT '{}',            -- เก็บ Keyword เป็น Array เช่น ['ข้าว', 'ใบเหลือง', 'เชื้อรา']
    ai_answer TEXT NOT NULL,                 -- คำตอบ/วิธีแก้ปัญหาจาก AI
    question_embedding VECTOR(768),          -- (Optional) เก็บ Vector ของคำถามสำหรับ Semantic Search (Gemini ใช้ 768 dimensions)
    is_verified BOOLEAN DEFAULT false,       -- Flag เผื่ออนาคตให้แอดมินมากด "ยืนยันความถูกต้อง" เพื่อใช้เป็น Knowledge base เกรด A
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. สร้าง GIN Index เพื่อให้ค้นหา Keyword ใน Array ได้เร็วระดับมิลลิวินาที
CREATE INDEX idx_consultations_keywords ON public.consultations USING GIN (keywords);

-- 4. เปิด RLS (Row Level Security) ป้องกันคนนอกยิง API ลบข้อมูลเล่น
ALTER TABLE public.consultations ENABLE ROW LEVEL SECURITY;

-- 5. กำหนด Policy เบื้องต้น (ให้ทุกคนอ่านและเพิ่มข้อมูลได้ แต่ลบ/แก้ไขไม่ได้)
CREATE POLICY "Allow public select" ON public.consultations FOR SELECT USING (true);
CREATE POLICY "Allow public insert" ON public.consultations FOR INSERT WITH CHECK (true);
