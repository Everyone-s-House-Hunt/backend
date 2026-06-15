-- テスト用シードデータ: イノシシパニック

-- シード用ユーザー（creator_user_id の外部キー制約を満たすため）
INSERT INTO users (id, username, email, password_hash, is_premium, created_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'seed_creator', 'seed@example.com', 'x', false, NOW(3))
ON DUPLICATE KEY UPDATE username = username;

-- イノシシパニックの問題（status = approved を12問）
INSERT INTO questions (id, creator_user_id, body, answer_data, explanation, game_mode, difficulty, status, created_at) VALUES
('q-inoshishi-001', '00000000-0000-0000-0000-000000000001', '日本の首都はどっち？',           '{"choices": ["東京", "大阪"], "correct_index": 0}', '日本の首都は東京です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-002', '00000000-0000-0000-0000-000000000001', '富士山があるのはどっち？',         '{"choices": ["静岡", "北海道"], "correct_index": 0}', '富士山は静岡県と山梨県にまたがっています。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-003', '00000000-0000-0000-0000-000000000001', '1 + 1 はどっち？',                '{"choices": ["2", "3"], "correct_index": 0}', '1 + 1 = 2 です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-004', '00000000-0000-0000-0000-000000000001', '海に住んでいるのはどっち？',       '{"choices": ["イルカ", "ライオン"], "correct_index": 0}', 'イルカは海に住む動物です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-005', '00000000-0000-0000-0000-000000000001', '太陽が昇るのはどっち？',           '{"choices": ["西", "東"], "correct_index": 1}', '太陽は東から昇ります。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-006', '00000000-0000-0000-0000-000000000001', '水の化学式はどっち？',             '{"choices": ["CO2", "H2O"], "correct_index": 1}', '水の化学式はH2Oです。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-007', '00000000-0000-0000-0000-000000000001', 'より大きい数はどっち？',           '{"choices": ["100", "99"], "correct_index": 0}', '100は99より大きい数です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-008', '00000000-0000-0000-0000-000000000001', '空を飛べるのはどっち？',           '{"choices": ["ペンギン", "ワシ"], "correct_index": 1}', 'ワシは空を飛べる鳥です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-009', '00000000-0000-0000-0000-000000000001', '冬に降るのはどっち？',             '{"choices": ["雪", "花粉"], "correct_index": 0}', '冬には雪が降ります。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-010', '00000000-0000-0000-0000-000000000001', '赤信号はどっち？',                 '{"choices": ["進め", "止まれ"], "correct_index": 1}', '赤信号は「止まれ」を意味します。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-011', '00000000-0000-0000-0000-000000000001', '甘いのはどっち？',                 '{"choices": ["砂糖", "塩"], "correct_index": 0}', '砂糖は甘い調味料です。', 'inoshishi_panic', 1, 'approved', NOW(3)),
('q-inoshishi-012', '00000000-0000-0000-0000-000000000001', '日本で一番高い山はどっち？',       '{"choices": ["富士山", "高尾山"], "correct_index": 0}', '日本で一番高い山は富士山です。', 'inoshishi_panic', 1, 'approved', NOW(3))
ON DUPLICATE KEY UPDATE body = VALUES(body);
