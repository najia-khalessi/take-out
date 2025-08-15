#!/bin/bash

# take-out外卖系统 - 消息系统与群聊功能测试脚本
# 测试订单相关的群聊消息功能

BASE_URL="http://localhost:8080"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 创建日志目录
mkdir -p test_logs/
exec 1> >(tee -a test_logs/messaging.log)
exec 2> >(tee -a test_logs/messaging_error.log >&2)

echo "======================================"
echo "开始测试消息系统功能"
echo "日期: $(date)"
echo "======================================"

# 检查依赖
if [ ! -f "test_data/user_token.txt" ] || [ ! -f "test_data/shop_token.txt" ] || [ ! -f "test_data/rider_token.txt" ]; then
    echo -e "${RED}✗ 未找到必要的token文件，请先运行 test_auth.sh${NC}"
    exit 1
fi

if [ ! -f "test_data/order_id.txt" ]; then
    echo -e "${RED}✗ 未找到订单ID，请先运行 test_order_flow.sh${NC}"
    exit 1
fi

# 加载token和订单信息
USER_TOKEN=$(cat test_data/user_token.txt)
SHOP_TOKEN=$(cat test_data/shop_token.txt)
RIDER_TOKEN=$(cat test_data/rider_token.txt)
ORDER_ID=$(cat test_data/order_id.txt)

# 从之前的测试数据获取商家ID
SHOP_ID=$(cat test_data/nearby_shops.json | jq -r '.[0].shop_id // .shops[0].shop_id' 2>/dev/null || echo "1")

# Step 1: 获取群聊历史消息
echo -e "${GREEN}步骤 1: 获取群聊历史消息${NC}"
GROUP_MESSAGES_USER=$(curl -s -X GET $BASE_URL/api/user/im/messages?group_id=$ORDER_ID \
  -H "Authorization: Bearer $USER_TOKEN")

echo "用户获取群聊消息: $GROUP_MESSAGES_USER"
echo "$GROUP_MESSAGES_USER" > test_data/group_messages_initial.json

# Step 2: 用户发送第一条消息
echo -e "${GREEN}步骤 2: 用户发送第一条消息${NC}"
USER_MESSAGE1=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"我刚下单了宫保鸡丁×2，麻烦不要放香菜，少油，谢谢！预计什么时候可以送达？\",
    \"sender_role\": \"user\"
  }")

echo "用户第一条消息: $USER_MESSAGE1"
echo "$USER_MESSAGE1" > test_data/user_message1.json

# Step 3: 商家回复消息
echo -e "${GREEN}步骤 3: 商家回复消息${NC}"
SHOP_MESSAGE1=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $SHOP_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"您好！订单已收到，正在准备您的宫保鸡丁。我们会按您的要求不加香菜少油，预计15分钟后可以出餐。\",
    \"sender_role\": \"shop\"
  }")

echo "商家回复: $SHOP_MESSAGE1"
echo "$SHOP_MESSAGE1" > test_data/shop_message1.json

# Step 4: 商家补充信息
echo -e "${GREEN}步骤 4: 商家通知出餐完成${NC}"
SHOP_MESSAGE2=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $SHOP_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"您的订单已制作完成！现在可以开始配送。菜品已按您的要求调整：无香菜、少油。\",
    \"sender_role\": \"shop\"
  }")

echo "商家出餐通知: $SHOP_MESSAGE2"
echo "$SHOP_MESSAGE2" > test_data/shop_message2.json

# Step 5: 骑手通知配送中
echo -e "${GREEN}步骤 5: 骑手通知配送中${NC}"
RIDER_MESSAGE1=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"您好！我是您的骑手，现已取餐出发，预计20分钟内到达海淀区中关村大街1号。注意接听电话！\",
    \"sender_role\": \"rider\"
  }")

echo "骑手出发通知: $RIDER_MESSAGE1"
echo "$RIDER_MESSAGE1" > test_data/rider_message1.json

# Step 6: 实时位置更新消息
echo -e "${GREEN}步骤 6: 骑手发送位置更新${NC}"
RIDER_MESSAGE2=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"距离您还有大概3公里，预计10分钟内到达。已经到达中关村南大街，很快了！\",
    \"sender_role\": \"rider\"
  }")

echo "骑手位置更新: $RIDER_MESSAGE2"
echo "$RIDER_MESSAGE2" > test_data/rider_message2.json

# Step 7: 用户追问
echo -e "${GREEN}步骤 7: 用户额外询问${NC}"
USER_MESSAGE2=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"好的，我已经到小区门口了，请问到哪里取餐？\",
    \"sender_role\": \"user\"
  }")

echo "用户追问: $USER_MESSAGE2"
echo "$USER_MESSAGE2" > test_data/user_message2.json

# Step 8: 骑手回复取餐
echo -e "${GREEN}步骤 8: 骑手回复取餐细节${NC}"
RIDER_MESSAGE3=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"已到达小区门口，我身穿蓝色外套，骑黄色外卖车后座有红色保温箱，请您出来取餐。\",
    \"sender_role\": \"rider\"
  }")

echo "骑手取餐详情: $RIDER_MESSAGE3"
echo "$RIDER_MESSAGE3" > test_data/rider_message3.json

# Step 9: 配送完成确认
echo -e "${GREEN}步骤 9: 配送完成后确认${NC}"
RIDER_MESSAGE4=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"订单已成功送达！感谢您使用我们的服务，希望您用餐愉快。如有任何问题，请及时联系外卖平台。\",
    \"sender_role\": \"rider\"
  }")

echo "配送完成消息: $RIDER_MESSAGE4"
echo "$RIDER_MESSAGE4" > test_data/rider_message4.json

# Step 10: 用户感谢回复
echo -e "${GREEN}步骤 10: 用户感谢回复${NC}"
USER_MESSAGE3=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"group_id\": $ORDER_ID,
    \"content\": \"收到！菜品很新鲜，配送很及时，谢谢各位的服务！\",
    \"sender_role\": \"user\"
  }")

echo "用户感谢: $USER_MESSAGE3"
echo "$USER_MESSAGE3" > test_data/user_message3.json

# Step 11: 测试获取所有聊天历史
echo -e "${GREEN}步骤 11: 获取完整聊天记录${NC}"
COMPLETE_MESSAGES=$(curl -s -X GET "$BASE_URL/api/user/im/messages?group_id=$ORDER_ID" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$COMPLETE_MESSAGES" > test_data/complete_chat_history.json

# Step 12: 验证消息功能
echo -e "${GREEN}步骤 12: 消息系统验证${NC}"
MESSAGE_COUNT=$(echo $COMPLETE_MESSAGES | jq '. | length' 2>/dev/null || echo "0")
echo "群内总消息数: $MESSAGE_COUNT"

# 统计各角色消息数
USER_MSGS=$(echo $COMPLETE_MESSAGES | jq '[.[] | select(.sender_name=="user")] | length' 2>/dev/null || echo "0")
SHOP_MSGS=$(echo $COMPLETE_MESSAGES | jq '[.[] | select(.sender_name=="shop")] | length' 2>/dev/null || echo "0")
RIDER_MSGS=$(echo $COMPLETE_MESSAGES | jq '[.[] | select(.sender_name=="rider")] | length' 2>/dev/null || echo "0")

echo "用户消息数: $USER_MSGS"
echo "商家消息数: $SHOP_MSGS"
echo "骑手消息数: $RIDER_MSGS"

# 测试批量发送消息
if command -v jq >/dev/null 2>&1; then
    echo -e "${GREEN}步骤 13: 消息性能测试${NC}"
    echo "开始测试短时间内发送多条消息的性能..."
    
    for i in {1..5}; do
        curl -s -X POST $BASE_URL/api/user/im/send \
          -H "Authorization: Bearer $USER_TOKEN" \
          -H "Content-Type: application/json" \
          -d "{
            \"group_id\": $ORDER_ID,
            \"content\": \"性能测试消息 $i / 5\",
            \"sender_role\": \"user\"
          }" > /dev/null
    done
    
    echo -e "${GREEN}✓ 批量消息发送完成${NC}"
fi

# 测试错误场景
echo -e "${GREEN}步骤 14: 错误场景测试${NC}"
echo "测试向不存在的群组发送消息..."
INVALID_GROUP_RESPONSE=$(curl -s -X POST $BASE_URL/api/user/im/send \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": 99999,
    "content": "测试非存在群组消息",
    "sender_role": "user"
  }' 2>/dev/null || echo '{"error":"invalid group"}')

echo "非存在群组响应: $INVALID_GROUP_RESPONSE"

# 测试无效Token
INVALID_TOKEN_RESPONSE=$(curl -s -X GET "$BASE_URL/api/user/im/messages?group_id=$ORDER_ID" \
  -H "Authorization: Bearer invalid_token_here" 2>/dev/null || echo '{"error":"invalid token"}')

echo "无效Token响应: $INVALID_TOKEN_RESPONSE"

# 总结报告
echo "======================================"
echo "消息系统测试完成总结"
echo "======================================"
echo "群聊ID: $ORDER_ID"
echo "测试消息角色组合:"
echo -e "${BLUE}User Role:\n✓ 发送询问消息n✓ 回复确认消息n✓ 发送感谢消息${NC}"
echo -e "${YELLOW}Shop Role:\n✓ 回复订单确认
✓ 通知出餐完成${NC}"
echo -e "${GREEN}Rider Role:\n✓ 配送开始通知
✓ 位置更新通知
✓ 送达确认通知${NC}"
echo ""
echo "测试数据统计:"
echo "总消息数: $MESSAGE_COUNT"
echo "用户消息: $USER_MSGS"
echo "商家消息: $SHOP_MSGS"  
echo "骑手消息: $RIDER_MSGS"
echo ""
echo -e "${GREEN}所有测试数据已保存到 test_data/ 目录${NC}"
echo -e "${GREEN}详细日志请查看 test_logs/messaging.log${NC}"
echo "======================================"