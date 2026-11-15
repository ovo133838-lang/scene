-- 题目01 电商订单交易系统 · 预置种子数据（幂等：固定 id + INSERT IGNORE，不覆盖运营后续修改）
USE shop_t01;

-- 预置账号（password_hash 占位，T2 由 auth.SeedUsers 用 bcrypt 重新初始化）
INSERT INTO `user` (username, password_hash, role) VALUES
('buyer1', 'PLACEHOLDER', 'buyer'),
('buyer2', 'PLACEHOLDER', 'buyer'),
('buyer3', 'PLACEHOLDER', 'buyer'),
('operator', 'PLACEHOLDER', 'operator'),
('warehouse', 'PLACEHOLDER', 'warehouse'),
('service', 'PLACEHOLDER', 'service')
ON DUPLICATE KEY UPDATE role = VALUES(role);

-- 预置商品：商品A 10000 分/可售 5；商品B 8000 分/可售 10（固定 id，幂等，不覆盖改价/库存）
INSERT IGNORE INTO product (id, name, price, stock, status) VALUES
(1, 'XX 品牌无线蓝牙耳机（标准款）', 10000, 5, 'on'),
(2, 'YY 便携式蓝牙音箱', 8000, 10, 'on');

-- 预置角色权限（RBAC 固化，固定 id，幂等）
INSERT IGNORE INTO role_permission (id, role, resource, action) VALUES
(1, 'operator', 'product', 'read'),
(2, 'operator', 'product', 'shelf'),
(3, 'operator', 'product', 'update'),
(4, 'operator', 'summary', 'read'),
(5, 'warehouse', 'shipment', 'read'),
(6, 'warehouse', 'shipment', 'ship'),
(7, 'service', 'refund', 'approve'),
(8, 'service', 'refund', 'reject'),
(9, 'service', 'refund', 'retry'),
(10, 'service', 'payment_abnormal', 'read'),
(11, 'service', 'payment_abnormal', 'refund');
