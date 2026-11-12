-- ชนิดพืช/สัตว์ที่ AI ระบุได้
ALTER TABLE public.consultations
    ADD COLUMN IF NOT EXISTS species         TEXT,  -- ชื่อกลางที่ใช้จับคู่ cache เช่น 'ข้าว', 'ทุเรียน', 'วัว'
    ADD COLUMN IF NOT EXISTS species_name    TEXT,  -- ชื่อที่ AI ระบุ (ละเอียดกว่า) เช่น 'ทุเรียนหมอนทอง'
    ADD COLUMN IF NOT EXISTS scientific_name TEXT;  -- เช่น 'Durio zibethinus'

CREATE INDEX IF NOT EXISTS idx_consultations_species ON public.consultations (species);
