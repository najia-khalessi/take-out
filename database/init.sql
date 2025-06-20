-- 创建数据库
DROP DATABASE IF EXISTS takeout;
CREATE DATABASE takeout CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE takeout;

-- 1. 用户表
CREATE TABLE users (
    userid INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    userpassword VARCHAR(255) NOT NULL COMMENT '用户密码',
    userphone VARCHAR(20) COMMENT '用户手机号',
    useraddress TEXT COMMENT '用户地址',
    userlatitude DECIMAL(10, 8) COMMENT '用户纬度',
    userlongitude DECIMAL(11, 8) COMMENT '用户经度',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) COMMENT='用户表';

-- 2. 商家表
CREATE TABLE shops (
    shopid INT AUTO_INCREMENT PRIMARY KEY,
    shopname VARCHAR(100) NOT NULL COMMENT '商家名称',
    shoppassword VARCHAR(255) NOT NULL COMMENT '商家密码',
    shopphone VARCHAR(20) COMMENT '商家联系电话',
    shopaddress TEXT COMMENT '商家地址',
    shopdescription TEXT COMMENT '商家描述',
    shoplatitude DECIMAL(10, 8) COMMENT '商家纬度',
    shoplongitude DECIMAL(11, 8) COMMENT '商家经度',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) COMMENT='商家表';

-- 3. 骑手表
CREATE TABLE riders (
    riderid INT AUTO_INCREMENT PRIMARY KEY,
    ridername VARCHAR(50) NOT NULL COMMENT '骑手姓名',
    riderpassword VARCHAR(255) NOT NULL COMMENT '骑手密码',
    riderstatus ENUM('online', 'offline', 'busy') DEFAULT 'offline' COMMENT '骑手状态：在线/离线/忙碌',
    riderlatitude DECIMAL(10, 8) COMMENT '骑手当前纬度',
    riderlongitude DECIMAL(11, 8) COMMENT '骑手当前经度',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) COMMENT='骑手表';

-- 4. 商品表
CREATE TABLE products (
    productid INT AUTO_INCREMENT PRIMARY KEY,
    shopid INT NOT NULL COMMENT '所属商家ID',
    productname VARCHAR(100) NOT NULL COMMENT '商品名称',
    productprice DECIMAL(10, 2) NOT NULL COMMENT '商品价格',
    prodescription TEXT COMMENT '商品描述',
    stock INT DEFAULT 0 COMMENT '库存数量',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE
) COMMENT='商品表';

-- 5. 订单表
CREATE TABLE orders (
    orderid INT AUTO_INCREMENT PRIMARY KEY,
    userid INT NOT NULL COMMENT '用户ID',
    shopid INT NOT NULL COMMENT '商家ID',
    riderid INT DEFAULT NULL COMMENT '骑手ID',
    productid INT NOT NULL COMMENT '商品ID',
    orderstatus ENUM('pending', 'confirmed', 'preparing', 'delivering', 'completed', 'cancelled') 
        DEFAULT 'pending' COMMENT '订单状态',
    username VARCHAR(50) COMMENT '用户名（冗余字段）',
    shopname VARCHAR(100) COMMENT '商家名（冗余字段）',
    ordertime TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '下单时间',
    productname VARCHAR(100) COMMENT '商品名（冗余字段）',
    totalprice DECIMAL(10, 2) NOT NULL COMMENT '订单总价',
    groupid INT DEFAULT NULL COMMENT '聊天群组ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE,
    FOREIGN KEY (riderid) REFERENCES riders(riderid) ON DELETE SET NULL,
    FOREIGN KEY (productid) REFERENCES products(productid) ON DELETE CASCADE
) COMMENT='订单表';

-- 6. 聊天群组表
CREATE TABLE groups (
    groupid INT AUTO_INCREMENT PRIMARY KEY,
    orderid INT NOT NULL COMMENT '关联订单ID',
    userid INT NOT NULL COMMENT '用户ID',
    shopid INT NOT NULL COMMENT '商家ID',
    riderid INT DEFAULT NULL COMMENT '骑手ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (orderid) REFERENCES orders(orderid) ON DELETE CASCADE,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE,
    FOREIGN KEY (riderid) REFERENCES riders(riderid) ON DELETE SET NULL
) COMMENT='聊天群组表';

-- 7. 消息表
CREATE TABLE messages (
    messageid INT AUTO_INCREMENT PRIMARY KEY,
    groupid INT NOT NULL COMMENT '群组ID',
    riderid INT DEFAULT NULL COMMENT '骑手ID',
    userid INT DEFAULT NULL COMMENT '用户ID', 
    shopid INT DEFAULT NULL COMMENT '商家ID',
    content TEXT NOT NULL COMMENT '消息内容',
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '消息时间戳',
    FOREIGN KEY (groupid) REFERENCES groups(groupid) ON DELETE CASCADE,
    FOREIGN KEY (riderid) REFERENCES riders(riderid) ON DELETE SET NULL,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE SET NULL,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE SET NULL
) COMMENT='消息表';

-- 创建索引优化查询性能
-- 用户表索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_userphone ON users(userphone);
CREATE INDEX idx_users_location ON users(userlatitude, userlongitude);

-- 商家表索引
CREATE INDEX idx_shops_shopname ON shops(shopname);
CREATE INDEX idx_shops_location ON shops(shoplatitude, shoplongitude);

-- 骑手表索引
CREATE INDEX idx_riders_status ON riders(riderstatus);
CREATE INDEX idx_riders_location ON riders(riderlatitude, riderlongitude);

-- 商品表索引
CREATE INDEX idx_products_shopid ON products(shopid);
CREATE INDEX idx_products_name ON products(productname);

-- 订单表索引
CREATE INDEX idx_orders_userid ON orders(userid);
CREATE INDEX idx_orders_shopid ON orders(shopid);
CREATE INDEX idx_orders_riderid ON orders(riderid);
CREATE INDEX idx_orders_status ON orders(orderstatus);
CREATE INDEX idx_orders_time ON orders(ordertime);

-- 消息表索引
CREATE INDEX idx_messages_groupid ON messages(groupid);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);

-- 插入一些测试数据
INSERT INTO users (username, userpassword, userphone, useraddress, userlatitude, userlongitude) VALUES
('testuser1', '$2a$14$example_hash_password', '13800138001', '北京市朝阳区', 39.9042, 116.4074),
('testuser2', '$2a$14$example_hash_password', '13800138002', '上海市浦东新区', 31.2304, 121.4737);

INSERT INTO shops (shopname, shoppassword, shopphone, shopaddress, shopdescription, shoplatitude, shoplongitude) VALUES
('麦当劳', '$2a$14$example_hash_password', '400-123-4567', '北京市朝阳区建国门外大街1号', '快餐连锁店', 39.9042, 116.4074),
('肯德基', '$2a$14$example_hash_password', '400-765-4321', '上海市浦东新区陆家嘴环路1000号', '快餐连锁店', 31.2304, 121.4737);

INSERT INTO riders (ridername, riderpassword, riderstatus, riderlatitude, riderlongitude) VALUES
('张三', '$2a$14$example_hash_password', 'online', 39.9042, 116.4074),
('李四', '$2a$14$example_hash_password', 'offline', 31.2304, 121.4737);
