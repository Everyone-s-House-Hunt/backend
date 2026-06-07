-- テスト用シードデータ: イノシシパニック

-- シード用ユーザー（creator_user_id の外部キー制約を満たすため）
INSERT INTO users (id, username, email, password_hash, is_premium, created_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'seed_creator', 'seed@example.com', 'x', false, NOW(3))
ON DUPLICATE KEY UPDATE username = username;

-- イノシシパニックの問題（status = approved を12問）
INSERT INTO questions (id, creator_user_id, body, answer_data, game_mode, status, created_at) VALUES
('q-inoshishi-001', '00000000-0000-0000-0000-000000000001', '日本の首都はどっち？',           '{"choices": ["東京", "大阪"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-002', '00000000-0000-0000-0000-000000000001', '富士山があるのはどっち？',         '{"choices": ["静岡", "北海道"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-003', '00000000-0000-0000-0000-000000000001', '1 + 1 はどっち？',                '{"choices": ["2", "3"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-004', '00000000-0000-0000-0000-000000000001', '海に住んでいるのはどっち？',       '{"choices": ["イルカ", "ライオン"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-005', '00000000-0000-0000-0000-000000000001', '太陽が昇るのはどっち？',           '{"choices": ["西", "東"], "correct_index": 1}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-006', '00000000-0000-0000-0000-000000000001', '水の化学式はどっち？',             '{"choices": ["CO2", "H2O"], "correct_index": 1}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-007', '00000000-0000-0000-0000-000000000001', 'より大きい数はどっち？',           '{"choices": ["100", "99"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-008', '00000000-0000-0000-0000-000000000001', '空を飛べるのはどっち？',           '{"choices": ["ペンギン", "ワシ"], "correct_index": 1}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-009', '00000000-0000-0000-0000-000000000001', '冬に降るのはどっち？',             '{"choices": ["雪", "花粉"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-010', '00000000-0000-0000-0000-000000000001', '赤信号はどっち？',                 '{"choices": ["進め", "止まれ"], "correct_index": 1}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-011', '00000000-0000-0000-0000-000000000001', '甘いのはどっち？',                 '{"choices": ["砂糖", "塩"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3)),
('q-inoshishi-012', '00000000-0000-0000-0000-000000000001', '日本で一番高い山はどっち？',       '{"choices": ["富士山", "高尾山"], "correct_index": 0}', 'inoshishi_panic', 'approved', NOW(3))
ON DUPLICATE KEY UPDATE body = VALUES(body);
