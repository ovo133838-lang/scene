-- 题目01 电商订单交易系统 · 预置种子数据
USE shop_t01;

-- 预置账号（password_hash 为占位，T2 鉴权阶段用 bcrypt 重新初始化）
-- 演示账号密码同用户名：buyer1/buyer2/buyer3/operator/warehouse/service
INSERT INTO `user` (username, password_hash, role) VALUES
('buyer1', 'PLACEHOLDER', 'buyer'),
('buyer2', 'PLACEHOLDER', 'buyer'),
('buyer3', 'PLACEHOLDER', 'buyer'),
('operator', 'PLACEHOLDER', 'operator'),
('warehouse', 'PLACEHOLDER', 'warehouse'),
('service', 'PLACEHOLDER', 'service')
ON DUPLICATE KEY UPDATE role = VALUES(role);

-- 预置商品：商品A 10000 分/可售 5；商品B 8000 分/可售 10
INSERT INTO product (name, price, stock, status) VALUES
('XX 品牌无线蓝牙耳机（标准款）', 10000, 5, 'on'),
('YY 便携式蓝牙音箱', 8000, 10, 'on');

-- 预置角色权限（RBAC 固化）
INSERT INTO role_permission (role, resource, action) VALUES
('operator', 'product', 'read'),
('operator', 'product', 'shelf'),
('operator', 'product', 'update'),
('operator', 'summary', 'read'),
('warehouse', 'shipment', 'read'),
('warehouse', 'shipment', 'ship'),
('service', 'refund', 'approve'),
('service', 'refund', 'reject'),
('service', 'refund', 'retry'),
('service', 'payment_abnormal', 'read'),
('service', 'payment_abnormal', 'refund');
