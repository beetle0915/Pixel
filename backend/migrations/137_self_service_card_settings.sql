INSERT INTO settings (key, value, created_at, updated_at)
VALUES
  ('self_service_card_enabled', 'true', NOW(), NOW()),
  ('self_service_card_label', '自主购卡', NOW(), NOW()),
  ('self_service_card_url', 'https://pay.ldxp.cn/shop/OTOKMA5D', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
