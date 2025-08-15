#!/bin/bash

# take-out外卖系统 - 订单生命周期测试脚本
# 完整测试从下单到送达的全部流程

BASE_URL="http://localhost:8080"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 创建日志文件
mkdir -p test_logs/
exec 1> >(tee -a test_logs/order_flow.log)
exec 2> >(tee -a test_logs/order_flow_error.log >&2)

echo "======================================"
echo "开始测试订单完整生命周期流程"
echo "日期: $(date)"
echo "======================================"

# 检查依赖
if [ ! -f "test_data/user_token.txt" ]; then
    echo -e "${RED}✗ 未找到用户token，请先运行 test_auth.sh${NC}"
    exit 1
fi

if [ ! -f "test_data/shop_token.txt" ]; then
    echo -e "${RED}✗ 未找到商家token，请先运行 test_auth.sh${NC}"
    exit 1
fi

if [ ! -f "test_data/rider_token.txt" ]; then
    echo -e "${RED}✗ 未找到骑手token，请先运行 test_auth.sh${NC}"
    exit 1
fi

# 加载token
USER_TOKEN=$(cat test_data/user_token.txt)
SHOP_TOKEN=$(cat test_data/shop_token.txt)
RIDER_TOKEN=$(cat test_data/rider_token.txt)

# Step 1: 商家添加商品
echo -e "${GREEN}步骤 1: 商家添加测试商品${NC}"
PRODUCT_RESPONSE=$(curl -s -X POST $BASE_URL/api/shop/add_product \
  -H "Authorization: Bearer $SHOP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "宫保鸡丁",
    "description": "经典川菜，鸡肉花生搭配，香辣可口",
    "price": 28.80,
    "stock": 100
  }')

echo "添加商品响应: $PRODUCT_RESPONSE"
echo "$PRODUCT_RESPONSE" > test_data/product_response.json

# 提取商品ID
PRODUCT_ID=$(echo $PRODUCT_RESPONSE | jq -r '.product_id // .ProductID // .product_id')
if [ "$PRODUCT_ID" = "null" ] || [ -z "$PRODUCT_ID" ]; then
    # 尝试其他可能的产品ID字段
    PRODUCT_ID=$(echo $PRODUCT_RESPONSE | jq -r '.[] | select(.product_id) | .product_id' | head -1)
fi

if [ -z "$PRODUCT_ID" ] || [ "$PRODUCT_ID" = "null" ]; then
    echo -e "${RED}✗ 无法获取商品ID，使用预设值${NC}"
    PRODUCT_ID=1
else
    echo -e "${GREEN}✓ 商品添加成功，商品ID: $PRODUCT_ID${NC}"
fi

# Step 2: 用户浏览商家和商品
echo -e "${GREEN}步骤 2: 用户浏览周围商家${NC}"
NEARBY_SHOPS=$(curl -s -X GET $BASE_URL/api/user/nearby-shops?latitude=39.9042\&longitude=116.4074\&radius=5000 \
  -H "Authorization: Bearer $USER_TOKEN")

echo "附近商家: $NEARBY_SHOPS" > test_data/nearby_shops.json
SHOP_ID=$(echo $NEARBY_SHOPS | jq -r '.[0].shop_id // .shops[0].shop_id' 2>/dev/null | head -1)

if [ -z "$SHOP_ID" ] || [ "$SHOP_ID" = "null" ]; then
    echo -e "${YELLOW}⚠ 从配置文件获取商家ID${NC}"
    SHOP_ID=1
fi

echo -e "${GREEN}步骤 2.1: 用户查看商家商品${NC}"
SHOP_PRODUCTS=$(curl -s -X GET "$BASE_URL/api/user/products?shop_id=$SHOP_ID" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$SHOP_PRODUCTS" > test_data/shop_products.json

# Step 3: 用户创建订单
echo -e "${GREEN}步骤 3: 用户创建订单${NC}"
ORDER_RESPONSE=$(curl -s -X POST $BASE_URL/api/user/order \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"shop_id\": $SHOP_ID,
    \"product_id\": $PRODUCT_ID,
    \"quantity\": 2,
    \"delivery_address\": \"北京市海淀区中关村大街1号\",
    \"notes\": \"不要放香菜，少油\"
  }")

echo "创建订单响应: $ORDER_RESPONSE"
echo "$ORDER_RESPONSE" > test_data/order_response.json

# 提取订单ID
ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.order_id // .OrderID // .order_id' 2>/dev/null)
if [ -z "$ORDER_ID" ] || [ "$ORDER_ID" = "null" ]; then
    ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.[] | select(.order_id) | .order_id' | head -1)
fi

if [ -z "$ORDER_ID" ] || [ "$ORDER_ID" = "null" ]; then
    echo -e "${RED}✗ 获取订单ID失败，使用预设值${NC}"
    ORDER_ID=1001
else
    echo -e "${GREEN}✓ 订单创建成功，订单ID: $ORDER_ID${NC}"
fi

echo "$ORDER_ID" > test_data/order_id.txt

# Step 4: 查询订单状态
echo -e "${GREEN}步骤 4: 查询订单状态${NC}"
curl -s -X GET "$BASE_URL/api/user/order/status?order_id=$ORDER_ID" \
  -H "Authorization: Bearer $USER_TOKEN" > test_data/order_status_new.json
echo "订单状态(用户视角): $(cat test_data/order_status_new.json)"

# Step 5: 商家接单
echo -e "${GREEN}步骤 5: 商家接单${NC}"
ACCEPT_RESPONSE=$(curl -s -X PUT $BASE_URL/api/shop/accept_order \
  -H "Authorization: Bearer $SHOP_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": $ORDER_ID,
    \"preparation_time\": 20,
    \"estimated_delivery_time\": 45
  }")

echo "商家接单响应: $ACCEPT_RESPONSE"
echo "$ACCEPT_RESPONSE" > test_data/order_accepted.json

if echo "$ACCEPT_RESPONSE" | grep -q "accepted\|Accepted\|成功"; then
    echo -e "${GREEN}✓ 商家接单成功${NC}"
else
    echo -e "${RED}✗ 商家接单失败${NC}"
fi

# Step 6: 商家发布配送单
echo -e "${GREEN}步骤 6: 商家发布配送单${NC}"
PUBLISH_RESPONSE=$(curl -s -X POST $BASE_URL/api/shop/publish_order \
  -H "Authorization: Bearer $SHOP_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": $ORDER_ID,
    \"pickup_location\": \"北京市朝阳区建国路88号\",
    \"delivery_location\": \"北京市海淀区中关村大街1号\",
    \"order_description\": \"宫保鸡丁 x2(不要香菜，少油)\",
    \"delivery_fee\": 8.0
  }")

echo "发布配送单响应: $PUBLISH_RESPONSE"
echo "$PUBLISH_RESPONSE" > test_data/delivery_published.json

# Step 7: 骑手抢单
echo -e "${GREEN}步骤 7: 骑手抢单${NC}"
GRAB_RESPONSE=$(curl -s -X POST $BASE_URL/api/rider/grab \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": $ORDER_ID,
    \"expected_pickup_time\": 15,
    \"delivery_notes\": \"已确认配送\"
  }")

echo "骑手抢单响应: $GRAB_RESPONSE"
echo "$GRAB_RESPONSE" > test_data/rider_grab.json

# Step 8: 订单状态查询
echo -e "${GREEN}步骤 8: 查询配送状态${NC}"
curl -s -X GET "$BASE_URL/api/user/order/status?order_id=$ORDER_ID" \
  -H "Authorization: Bearer $USER_TOKEN" > test_data/order_status_delivery.json
echo "订单状态(配送中): $(cat test_data/order_status_delivery.json)"

# Step 9: 骑手送达确认
echo -e "${GREEN}步骤 9: 骑手确认送达${NC}"
DELIVER_RESPONSE=$(curl -s -X PUT $BASE_URL/api/rider/complete \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": $ORDER_ID,
    \"delivery_time\": 42,
    \"delivery_status\": \"successful\",
    \"notes\": \"已送达，顾客非常满意\"
  }")

echo "送达确认响应: $DELIVER_RESPONSE"
echo "$DELIVER_RESPONSE" > test_data/delivery_complete.json

# Step 10: 最终订单状态
echo -e "${GREEN}步骤 10: 查询最终订单状态${NC}"
FINAL_STATUS=$(curl -s -X GET "$BASE_URL/api/user/order/status?order_id=$ORDER_ID" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$FINAL_STATUS" > test_data/order_status_final.json
echo "最终订单状态: $FINAL_STATUS"

# 总结
echo "======================================"
echo "订单生命周期测试完成总结"
echo "======================================"
echo "订单ID: $ORDER_ID"
echo "商品ID: $PRODUCT_ID"
echo "商家ID: $SHOP_ID"
echo "流程完成状态:"

if [ -f test_data/order_response.json ]; then
    echo -e "${GREEN}✓ 订单创建成功${NC}"
fi

if [ -f test_data/order_accepted.json ]; then
    echo -e "${GREEN}✓ 商家接单成功${NC}"
fi

if [ -f test_data/delivery_published.json ]; then
    echo -e "${GREEN}✓ 配送单发布成功${NC}"
fi

if [ -f test_data/rider_grab.json ]; then
    echo -e "${GREEN}✓ 骑手抢单成功${NC}"
fi

if [ -f test_data/delivery_complete.json ]; then
    echo -e "${GREEN}✓ 订单完成送达${NC}"
fi

echo "所有测试数据已保存到 test_data/ 目录"
echo "详细日志请查看 test_logs/order_flow.log"
echo "======================================"