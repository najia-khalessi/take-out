CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

-- 自动更新 'updated_at' 列的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 1. 用户表
CREATE TABLE users (
    userid SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    userpassword VARCHAR(255) NOT NULL,
    userphone VARCHAR(20),
    useraddress TEXT,
    userlatitude DECIMAL(10, 8),
    userlongitude DECIMAL(11, 8),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users.username IS '用户名';
COMMENT ON COLUMN users.userpassword IS '用户密码';
COMMENT ON COLUMN users.userphone IS '用户手机号';
COMMENT ON COLUMN users.useraddress IS '用户地址';
COMMENT ON COLUMN users.userlatitude IS '用户纬度';
COMMENT ON COLUMN users.userlongitude IS '用户经度';
COMMENT ON COLUMN users.created_at IS '创建时间';
COMMENT ON COLUMN users.updated_at IS '更新时间';

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 2. 商家表
CREATE TABLE shops (
    shopid SERIAL PRIMARY KEY,
    shopname VARCHAR(100) NOT NULL UNIQUE,
    shoppassword VARCHAR(255) NOT NULL,
    shopphone VARCHAR(20),
    shopaddress TEXT,
    shopdescription TEXT,
    shoplatitude DECIMAL(10, 8),
    shoplongitude DECIMAL(11, 8),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE shops IS '商家表';
COMMENT ON COLUMN shops.shopname IS '商家名称';
COMMENT ON COLUMN shops.shoppassword IS '商家密码';
COMMENT ON COLUMN shops.shopphone IS '商家联系电话';
COMMENT ON COLUMN shops.shopaddress IS '商家地址';
COMMENT ON COLUMN shops.shopdescription IS '商家描述';
COMMENT ON COLUMN shops.shoplatitude IS '商家纬度';
COMMENT ON COLUMN shops.shoplongitude IS '商家经度';
COMMENT ON COLUMN shops.created_at IS '创建时间';
COMMENT ON COLUMN shops.updated_at IS '更新时间';

CREATE TRIGGER update_shops_updated_at
BEFORE UPDATE ON shops
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 3. 骑手表
CREATE TABLE riders (
    riderid SERIAL PRIMARY KEY,
    ridername VARCHAR(50) NOT NULL UNIQUE,
    riderpassword VARCHAR(255) NOT NULL,
    riderphone VARCHAR(20),
    vehicletype VARCHAR(50),
    riderstatus VARCHAR(10) DEFAULT 'offline' CHECK (riderstatus IN ('online', 'offline', 'busy')),
    rating DECIMAL(3, 2) DEFAULT 5.00,
    riderlatitude DECIMAL(10, 8),
    riderlongitude DECIMAL(11, 8),
    delivery_fee DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE riders IS '骑手表';
COMMENT ON COLUMN riders.ridername IS '骑手用户名';
COMMENT ON COLUMN riders.riderpassword IS '骑手密码';
COMMENT ON COLUMN riders.riderphone IS '骑手手机号';
COMMENT ON COLUMN riders.vehicletype IS '交通工具类型';
COMMENT ON COLUMN riders.riderstatus IS '骑手状态：在线/离线/忙碌';
COMMENT ON COLUMN riders.rating IS '骑手评分';
COMMENT ON COLUMN riders.riderlatitude IS '骑手当前纬度';
COMMENT ON COLUMN riders.riderlongitude IS '骑手当前经度';
COMMENT ON COLUMN riders.delivery_fee IS '配送费';
COMMENT ON COLUMN riders.created_at IS '创建时间';
COMMENT ON COLUMN riders.updated_at IS '更新时间';

-- 添加触发器
CREATE TRIGGER update_riders_updated_at
BEFORE UPDATE ON riders
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 添加索引
CREATE INDEX idx_riders_status ON riders(riderstatus);
CREATE INDEX idx_riders_location ON riders(riderlatitude, riderlongitude);
CREATE INDEX idx_riders_riderphone ON riders(riderphone);

-- 4. 商品表
CREATE TABLE products (
    productid SERIAL PRIMARY KEY,
    shopid INT NOT NULL,
    productname VARCHAR(100) NOT NULL,
    productprice DECIMAL(10, 2) NOT NULL,
    prodescription TEXT,
    stock INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE
);

COMMENT ON TABLE products IS '商品表';
COMMENT ON COLUMN products.shopid IS '所属商家ID';
COMMENT ON COLUMN products.productname IS '商品名称';
COMMENT ON COLUMN products.productprice IS '商品价格';
COMMENT ON COLUMN products.prodescription IS '商品描述';
COMMENT ON COLUMN products.stock IS '库存数量';
COMMENT ON COLUMN products.created_at IS '创建时间';
COMMENT ON COLUMN products.updated_at IS '更新时间';

CREATE TRIGGER update_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 5. 订单表
CREATE TABLE orders (
    orderid SERIAL PRIMARY KEY,
    userid INT NOT NULL,
    shopid INT NOT NULL,
    riderid INT DEFAULT NULL,
    productid INT NOT NULL,
    orderstatus VARCHAR(20) DEFAULT 'pending' CHECK (orderstatus IN ('pending', 'confirmed', 'preparing', 'delivering', 'completed', 'cancelled')),
    username VARCHAR(50),
    shopname VARCHAR(100),
    ordertime TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    productname VARCHAR(100),
    totalprice DECIMAL(10, 2) NOT NULL,
    delivery_fee DECIMAL(10,2) DEFAULT 0,
    groupid INT DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE,
    FOREIGN KEY (riderid) REFERENCES riders(riderid) ON DELETE SET NULL,
    FOREIGN KEY (productid) REFERENCES products(productid) ON DELETE CASCADE
);

COMMENT ON TABLE orders IS '订单表';
COMMENT ON COLUMN orders.userid IS '用户ID';
COMMENT ON COLUMN orders.shopid IS '商家ID';
COMMENT ON COLUMN orders.riderid IS '骑手ID';
COMMENT ON COLUMN orders.productid IS '商品ID';
COMMENT ON COLUMN orders.orderstatus IS '订单状态';
COMMENT ON COLUMN orders.username IS '用户名（冗余字段）';
COMMENT ON COLUMN orders.shopname IS '商家名（冗余字段）';
COMMENT ON COLUMN orders.ordertime IS '下单时间';
COMMENT ON COLUMN orders.productname IS '商品名（冗余字段）';
COMMENT ON COLUMN orders.totalprice IS '订单总价';
COMMENT ON COLUMN orders.delivery_fee IS '配送费';
COMMENT ON COLUMN orders.groupid IS '聊天群组ID';
COMMENT ON COLUMN orders.created_at IS '创建时间';
COMMENT ON COLUMN orders.updated_at IS '更新时间';

CREATE TRIGGER update_orders_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 6. 聊天群组表
CREATE TABLE groups (
    groupid SERIAL PRIMARY KEY,
    orderid INT NOT NULL,
    userid INT NOT NULL,
    shopid INT NOT NULL,
    riderid INT DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (orderid) REFERENCES orders(orderid) ON DELETE CASCADE,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
    FOREIGN KEY (shopid) REFERENCES shops(shopid) ON DELETE CASCADE,
    FOREIGN KEY (riderid) REFERENCES riders(riderid) ON DELETE SET NULL
);

COMMENT ON TABLE groups IS '聊天群组表';
COMMENT ON COLUMN groups.orderid IS '关联订单ID';
COMMENT ON COLUMN groups.userid IS '用户ID';
COMMENT ON COLUMN groups.shopid IS '商家ID';
COMMENT ON COLUMN groups.riderid IS '骑手ID';
COMMENT ON COLUMN groups.created_at IS '创建时间';

-- 7. 消息表
CREATE TABLE messages (
    messageid SERIAL PRIMARY KEY,
    groupid INT NOT NULL,
    sender_id INT NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (groupid) REFERENCES groups(groupid) ON DELETE CASCADE
);

COMMENT ON TABLE messages IS '消息表';
COMMENT ON COLUMN messages.groupid IS '群组ID';
COMMENT ON COLUMN messages.sender_id IS '发送者ID (可以是user, shop, or rider)';
COMMENT ON COLUMN messages.sender_name IS '发送者名称';
COMMENT ON COLUMN messages.content IS '消息内容';
COMMENT ON COLUMN messages.timestamp IS '消息时间戳';

-- 创建索引优化查询性能
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_userphone ON users(userphone);
CREATE INDEX idx_users_location ON users(userlatitude, userlongitude);

CREATE INDEX idx_shops_shopname ON shops(shopname);
CREATE INDEX idx_shops_location ON shops(shoplatitude, shoplongitude);

CREATE INDEX idx_riders_status ON riders(riderstatus);
CREATE INDEX idx_riders_location ON riders(riderlatitude, riderlongitude);

CREATE INDEX idx_products_shopid ON products(shopid);
CREATE INDEX idx_products_name ON products(productname);

CREATE INDEX idx_orders_userid ON orders(userid);
CREATE INDEX idx_orders_shopid ON orders(shopid);
CREATE INDEX idx_orders_riderid ON orders(riderid);
CREATE INDEX idx_orders_status ON orders(orderstatus);
CREATE INDEX idx_orders_time ON orders(ordertime);

CREATE INDEX idx_messages_groupid ON messages(groupid);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);

-- 插入一些测试数据
INSERT INTO users (username, userpassword, userphone, useraddress, userlatitude, userlongitude) VALUES
('testuser1', 'hashed_password_placeholder', '13800138001', '北京市朝阳区', 39.9042, 116.4074),
('testuser2', 'hashed_password_placeholder', '13800138002', '上海市浦东新区', 31.2304, 121.4737);

INSERT INTO shops (shopname, shoppassword, shopphone, shopaddress, shopdescription, shoplatitude, shoplongitude) VALUES
('麦当劳', 'hashed_password_placeholder', '400-123-4567', '北京市朝阳区建国门外大街1号', '快餐连锁店', 39.9042, 116.4074),
('肯德基', 'hashed_password_placeholder', '400-765-4321', '上海市浦东新区陆家嘴环路1000号', '快餐连锁店', 31.2304, 121.4737);

-- 评价主表
CREATE TABLE reviews (
    review_id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(orderid) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(userid) ON DELETE CASCADE,
    shop_id INTEGER NOT NULL REFERENCES shops(shopid) ON DELETE CASCADE,
    rider_id INTEGER REFERENCES riders(riderid) ON DELETE SET NULL,

    -- 评价内容
    rating INTEGER CHECK (rating >= 1 AND rating <= 5) NOT NULL,
    content TEXT,

    -- AI分析字段
    sentiment_score DECIMAL(3,2) DEFAULT 0.00,
    sentiment_label VARCHAR(20) DEFAULT 'neutral',
    issue_categories TEXT[], -- JSON数组格式存储问题类别

    -- 商家回复
    shop_reply TEXT,
    replied_at TIMESTAMP,

    -- 自动评价标记
    is_auto_review BOOLEAN DEFAULT FALSE,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束：一个订单只能有一条评价
    UNIQUE(order_id)
);

-- AI分析结果表
CREATE TABLE ai_analysis (
    analysis_id SERIAL PRIMARY KEY,
    review_id INTEGER NOT NULL REFERENCES reviews(review_id) ON DELETE CASCADE,

    -- 情感分析
    emotional_intensity INTEGER CHECK (emotional_intensity >= 1 AND emotional_intensity <= 5),

    -- 问题分类
    delivery_issue BOOLEAN DEFAULT FALSE,
    food_quality_issue BOOLEAN DEFAULT FALSE,
    service_issue BOOLEAN DEFAULT FALSE,
    packaging_issue BOOLEAN DEFAULT FALSE,
    price_issue BOOLEAN DEFAULT FALSE,

    -- 摘要生成
    summary_20chars VARCHAR(20),
    suggested_reply TEXT,

    -- AI调用记录
    api_response JSONB,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_reviews_shop_id ON reviews(shop_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
CREATE INDEX idx_reviews_sentiment ON reviews(sentiment_label);
CREATE INDEX idx_reviews_created ON reviews(created_at);

-- 扩充订单状态
ALTER TABLE orders ADD COLUMN IF NOT EXISTS deliveryconfirmed_at TIMESTAMP;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS review_deadline TIMESTAMP;

-- 骑手确认送达状态
ALTER TABLE orders ADD COLUMN delivery_confirmed_by_rider BOOLEAN DEFAULT FALSE;

-- Token黑名单表
CREATE TABLE token_blacklist (
    jti VARCHAR(255) PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL
);
