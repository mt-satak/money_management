-- E2Eテスト用のテストデータ
-- パスワードはすべて 'password' でハッシュ化済み

-- データクリーンアップ
DELETE FROM bill_items;
DELETE FROM monthly_bills;
DELETE FROM users;

-- ユーザーデータの挿入
INSERT INTO users (name, account_id, password_hash, created_at, updated_at) VALUES
('テストユーザー1', 'testuser1', '$2a$10$mQJwwG8NweEwfKvaftyVyu8NkPCfsGT72gVsZZqg274uPzOg6CDa.', NOW(), NOW()),
('テストユーザー2', 'testuser2', '$2a$10$mQJwwG8NweEwfKvaftyVyu8NkPCfsGT72gVsZZqg274uPzOg6CDa.', NOW(), NOW()),
('山田太郎', 'yamada', '$2a$10$mQJwwG8NweEwfKvaftyVyu8NkPCfsGT72gVsZZqg274uPzOg6CDa.', NOW(), NOW());

-- 家計簿データの挿入
INSERT INTO monthly_bills (year, month, requester_id, payer_id, status, created_at, updated_at) VALUES
(2025, 8, 1, 2, 'pending', NOW(), NOW()),
(2025, 9, 2, 1, 'requested', NOW(), NOW());

-- 家計簿項目データの挿入
INSERT INTO bill_items (bill_id, item_name, amount, created_at, updated_at) VALUES
(1, 'スーパーでの買い物', 5000, NOW(), NOW()),
(1, 'ガソリン代', 3000, NOW(), NOW()),
(1, '電気代', 8000, NOW(), NOW()),
(1, 'インターネット料金', 4000, NOW(), NOW()),
(1, 'コンビニ', 1000, NOW(), NOW()),
(1, 'ドラッグストア', 2000, NOW(), NOW()),
(1, 'カフェ', 800, NOW(), NOW()),
(2, 'レストランでの食事', 12000, NOW(), NOW()),
(2, '映画チケット', 3600, NOW(), NOW());