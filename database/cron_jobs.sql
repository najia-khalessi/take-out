-- 设置自动好评
CREATE OR REPLACE FUNCTION create_auto_reviews()
RETURNS INTEGER AS $$
DECLARE
    affected_count INTEGER;
BEGIN
    -- 查找已完成订单且未评价的，超过72小时自动好评
    INSERT INTO reviews (order_id, user_id, shop_id, rider_id, rating, content, is_auto_review)
    SELECT
        o.orderid,
        o.userid,
        o.shopid,
        o.riderid,
        5,
        '系统自动好评',
        TRUE
    FROM orders o
    WHERE o.orderstatus = 'completed'
    AND NOT EXISTS (SELECT 1 FROM reviews r WHERE r.order_id = o.orderid)
    AND o.deliveryconfirmed_at < NOW() - INTERVAL '72 hours';

    GET DIAGNOSTICS affected_count = ROW_COUNT;
    RETURN affected_count;
END;
$$ LANGUAGE plpgsql;

-- 设置定时任务
SELECT cron.schedule('auto-review-task', '0 */6 * * *', 'SELECT create_auto_reviews()');
