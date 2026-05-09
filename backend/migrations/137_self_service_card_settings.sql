INSERT INTO settings (key, value)
VALUES
  ('self_service_card_enabled', 'true'),
  ('self_service_card_label', '自主购卡'),
  ('self_service_card_url', 'https://pay.ldxp.cn/shop/OTOKMA5D')
ON CONFLICT (key) DO NOTHING;
