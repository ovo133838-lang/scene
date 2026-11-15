-- 题目01 电商订单交易系统 · 数据库初始化 DDL
-- 依据 docs/02-数据字典.md，金额一律 BIGINT 存「分」

CREATE DATABASE IF NOT EXISTS shop_t01 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE shop_t01;

-- 1. 用户表
CREATE TABLE IF NOT EXISTS `user` (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  password_hash VARCHAR(128) NOT NULL,
  role VARCHAR(20) NOT NULL COMMENT 'buyer/operator/warehouse/service',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. 商品表
CREATE TABLE IF NOT EXISTS product (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  price BIGINT NOT NULL COMMENT '单价(分)',
  stock INT NOT NULL COMMENT '当前可售数量',
  status VARCHAR(10) NOT NULL DEFAULT 'off' COMMENT 'on/off',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. 订单表
CREATE TABLE IF NOT EXISTS `order` (
  id VARCHAR(32) NOT NULL COMMENT '订单号 T01+日期+序号',
  buyer_id BIGINT UNSIGNED NOT NULL,
  product_id BIGINT UNSIGNED NOT NULL,
  product_name VARCHAR(128) NOT NULL COMMENT '商品名快照',
  price_snapshot BIGINT NOT NULL COMMENT '下单单价快照(分)',
  quantity INT NOT NULL COMMENT '数量 1~2',
  total_amount BIGINT NOT NULL COMMENT '总金额(分)',
  status VARCHAR(20) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expired_at DATETIME NOT NULL COMMENT '待付款超时时间',
  updated_at DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_buyer (buyer_id),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. 支付流水表
CREATE TABLE IF NOT EXISTS payment (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id VARCHAR(32) NOT NULL,
  channel_txn_no VARCHAR(64) NOT NULL COMMENT '渠道交易号(全局唯一)',
  amount BIGINT NOT NULL COMMENT '金额(分)',
  status VARCHAR(20) NOT NULL COMMENT 'pending/success/failed',
  notify_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_channel_txn_no (channel_txn_no),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. 退款单表
CREATE TABLE IF NOT EXISTS refund (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id VARCHAR(32) NOT NULL,
  payment_id BIGINT UNSIGNED NULL,
  amount BIGINT NOT NULL COMMENT '退款金额(分)',
  type VARCHAR(20) NOT NULL COMMENT 'sale 售前 / late_payment 迟到支付',
  status VARCHAR(20) NOT NULL COMMENT 'pending/processing/success/failed',
  reason VARCHAR(255) NULL,
  retry_count INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 6. 发货单表
CREATE TABLE IF NOT EXISTS shipment (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id VARCHAR(32) NOT NULL,
  carrier VARCHAR(64) NOT NULL,
  tracking_no VARCHAR(64) NOT NULL,
  operator_id BIGINT UNSIGNED NOT NULL,
  shipped_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 7. 异常支付表
CREATE TABLE IF NOT EXISTS payment_abnormal (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id VARCHAR(32) NOT NULL,
  channel_txn_no VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL COMMENT '金额(分)',
  status VARCHAR(20) NOT NULL COMMENT 'pending/refunded',
  reason VARCHAR(255) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 8. 角色权限表
CREATE TABLE IF NOT EXISTS role_permission (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role VARCHAR(20) NOT NULL,
  resource VARCHAR(64) NOT NULL,
  action VARCHAR(32) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 9. 订单事件表（追溯）
CREATE TABLE IF NOT EXISTS order_event (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id VARCHAR(32) NOT NULL,
  event VARCHAR(32) NOT NULL,
  detail VARCHAR(255) NULL,
  operator_id BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
