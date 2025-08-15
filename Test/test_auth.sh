#!/bin/bash

# take-out外卖系统 - 认证系统测试脚本
# 包含用户、商家、骑手的注册与登录测试

BASE_URL="http://localhost:8080"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 测试用户数据
TEST_USER_PHONE="${1:-18612345678}"
TEST_USER_PWD="${2:-TestUser123}"
TEST_USER_NAME="${3:-测试用户}"
TEST_USER_ADDRESS="${4:-北京市海淀区中关村大街1号}"

TEST_SHOP_PHONE="${5:-18612345679}"
TEST_SHOP_PWD="${6:-TestShop123}"
TEST_SHOP_NAME="${7:-测试饭店}"
TEST_SHOP_ADDRESS="${8:-北京市朝阳区建国路88号}"

TEST_RIDER_PHONE="${9:-18612345680}"
TEST_RIDER_PWD="${10:-TestRider123}"
TEST_RIDER_NAME="${11:-测试骑手}"

echo "======================================"
echo "开始测试外卖系统认证功能"
echo "======================================"

# 创建测试数据目录
mkdir -p test_data/

# 1. 测试用户注册
echo -e "${GREEN}1. [用户注册]${NC}"
USER_REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/user/register \
  -H "Content-Type: application/json" \
  -d "{
    \"user_name\": \"$TEST_USER_NAME\",
    \"user_password\": \"$TEST_USER_PWD\",
    \"user_phone\": \"$TEST_USER_PHONE\",
    \"user_address\": \"$TEST_USER_ADDRESS\"
  }")

echo "用户注册响应: $USER_REGISTER_RESPONSE"
echo $USER_REGISTER_RESPONSE > test_data/user_register.json

# 2. 测试用户登录
echo -e "${GREEN}2. [用户登录]${NC}"
USER_LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/user/login \
  -H "Content-Type: application/json" \
  -d "{
    \"user_phone\": \"$TEST_USER_PHONE\",
    \"user_password\": \"$TEST_USER_PWD\"
  }")

echo "用户登录响应: $USER_LOGIN_RESPONSE"
echo $USER_LOGIN_RESPONSE > test_data/user_login.json

# 提取用户Token
USER_TOKEN=$(echo $USER_LOGIN_RESPONSE | jq -r '.access_token')
if [ "$USER_TOKEN" != "null" ] && [ "$USER_TOKEN" != "" ]; then
    echo -e "${GREEN}✓ 用户登录成功，Token已保存${NC}"
    echo $USER_TOKEN > test_data/user_token.txt
else
    echo -e "${RED}✗ 用户登录失败${NC}"
fi

# 3. 测试商家注册
echo -e "${GREEN}3. [商家注册]${NC}"
SHOP_REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/shop/register \
  -H "Content-Type: application/json" \
  -d "{
    \"shop_name\": \"$TEST_SHOP_NAME\",
    \"shop_password\": \"$TEST_SHOP_PWD\",
    \"shop_phone\": \"$TEST_SHOP_PHONE\",
    \"shop_address\": \"$TEST_SHOP_ADDRESS\",
    \"description\": \"这是一家测试饭店\",
    \"shop_latitude\": 39.9042,
    \"shop_longitude\": 116.4074
  }")

echo "商家注册响应: $SHOP_REGISTER_RESPONSE"
echo $SHOP_REGISTER_RESPONSE > test_data/shop_register.json

# 4. 测试商家登录
echo -e "${GREEN}4. [商家登录]${NC}"
SHOP_LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/shop/login \
  -H "Content-Type: application/json" \
  -d "{
    \"shop_phone\": \"$TEST_SHOP_PHONE\",
    \"shop_password\": \"$TEST_SHOP_PWD\"
  }")

echo "商家登录响应: $SHOP_LOGIN_RESPONSE"
echo $SHOP_LOGIN_RESPONSE > test_data/shop_login.json

# 提取商家Token
SHOP_TOKEN=$(echo $SHOP_LOGIN_RESPONSE | jq -r '.access_token')
if [ "$SHOP_TOKEN" != "null" ] && [ "$SHOP_TOKEN" != "" ]; then
    echo -e "${GREEN}✓ 商家登录成功，Token已保存${NC}"
    echo $SHOP_TOKEN > test_data/shop_token.txt
else
    echo -e "${RED}✗ 商家登录失败${NC}"
fi

# 5. 测试骑手注册
echo -e "${GREEN}5. [骑手注册]${NC}"
RIDER_REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/rider/register \
  -H "Content-Type: application/json" \
  -d "{
    \"rider_name\": \"$TEST_RIDER_NAME\",
    \"rider_password\": \"$TEST_RIDER_PWD\",
    \"rider_phone\": \"$TEST_RIDER_PHONE\",
    \"rider_latitude\": 39.9042,
    \"rider_longitude\": 116.4074,
    \"vehicle_type\": \"摩托车\",
    \"delivery_fee\": 5.0
  }")


echo "骑手注册响应: $RIDER_REGISTER_RESPONSE"
echo $RIDER_REGISTER_RESPONSE > test_data/rider_register.json

# 6. 测试骑手登录
echo -e "${GREEN}6. [骑手登录]${NC}"
RIDER_LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/rider/login \
  -H "Content-Type: application/json" \
  -d "{
    \"rider_phone\": \"$TEST_RIDER_PHONE\",
    \"rider_password\": \"$TEST_RIDER_PWD\"
  }")

echo "骑手登录响应: $RIDER_LOGIN_RESPONSE"
echo $RIDER_LOGIN_RESPONSE > test_data/rider_login.json

# 提取骑手Token
RIDER_TOKEN=$(echo $RIDER_LOGIN_RESPONSE | jq -r '.access_token')
if [ "$RIDER_TOKEN" != "null" ] && [ "$RIDER_TOKEN" != "" ]; then
    echo -e "${GREEN}✓ 骑手登录成功，Token已保存${NC}"
    echo $RIDER_TOKEN > test_data/rider_token.txt
else
    echo -e "${RED}✗ 骑手登录失败${NC}"
fi

# 7. 测试Token刷新
echo -e "${GREEN}7. [Token刷新]${NC}"
if [ -f test_data/user_token.txt ]; then
    REFRESH_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/refresh \
      -H "Content-Type: application/json" \
      -d "{
        \"refresh_token\": \"$(cat test_data/user_login.json | jq -r '.refresh_token')\"
      }")
    
    echo "Token刷新响应: $REFRESH_RESPONSE"
    echo $REFRESH_RESPONSE > test_data/refresh_token.json
else
    echo -e "${RED}✗ 未找到Token，跳过刷新测试${NC}"
fi

# 8. 验证Token有效性
echo -e "${GREEN}8. [Token验证]${NC}"
if [ -f test_data/user_token.txt ]; then
    VERIFY_RESPONSE=$(curl -s -X GET "$BASE_URL/api/user/nearby-shops?latitude=39.9042&longitude=116.4074" \
      -H "Authorization: Bearer $(cat test_data/user_token.txt)")
    
    echo "Token验证响应: $VERIFY_RESPONSE"
    if echo "$VERIFY_RESPONSE" | grep -q "shop" || echo "$VERIFY_RESPONSE" | grep -q "data"; then
        echo -e "${GREEN}✓ 用户Token有效${NC}"
    else
        echo -e "${RED}✗ 用户Token无效${NC}"
    fi
fi

echo "======================================"
echo "认证功能测试完成"
echo "测试数据已保存到 test_data/ 目录"
echo "======================================"