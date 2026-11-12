-- ชื่อพืช/สัตว์ภาษาอังกฤษ เช่น 'Rice', 'Durian', 'Dairy cow'
ALTER TABLE public.consultations
    ADD COLUMN IF NOT EXISTS english_name TEXT;
