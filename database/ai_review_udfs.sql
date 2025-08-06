-- 文件：database/ai_review_udfs.sql

-- 1. 初始化函数
CREATE OR REPLACE FUNCTION ai_review_init()
RETURNS VOID AS $$
BEGIN
    -- 安装必要扩展
    CREATE EXTENSION IF NOT EXISTS plpython3u;
    CREATE EXTENSION IF NOT EXISTS pgvector;

    -- 创建配置表
    CREATE TABLE IF NOT EXISTS ai_config (
        key VARCHAR(100) PRIMARY KEY,
        value TEXT
    );

    -- 插入DeepSeek配置
    INSERT INTO ai_config (key, value) VALUES
        ('deepseek_api_key', 'your-api-key-here'),
        ('api_timeout', '30'),
        ('sentiment_model', 'deepseek-chat');
END;
$$ LANGUAGE plpgsql;

-- 2. 主要分析函数
CREATE OR REPLACE FUNCTION ai_review_analyze(
    review_content TEXT,
    user_rating INTEGER
) RETURNS JSON AS $$
DECLARE
    result JSON;
    prompt TEXT;
BEGIN
    -- 构建prompt
    prompt := '作为外卖平台客服分析专家，请分析以下用户评价并提供结构化分析结果：' ||
              '\n评价内容：' || review_content ||
              '\n用户评分：' || user_rating || '/5' ||
              '\n请返回JSON格式：{"emotional_intensity":1-5,"delivery_issue":bool,"food_quality_issue":bool,"service_issue":bool,"summary_20chars":"20字摘要","suggested_reply":"商家回复建议"}';

    -- 调用DeepSeek API
    SELECT ai_call_deepseek(prompt) INTO result;

    RETURN result::JSON;
END;
$$ LANGUAGE plpgsql;

-- 3. 清理函数
CREATE OR REPLACE FUNCTION ai_review_deinit()
RETURNS VOID AS $$
BEGIN
    -- 清理临时数据
    TRUNCATE ai_analysis;
    UPDATE ai_config SET value = '' WHERE key = 'deepseek_api_key';
END;
$$ LANGUAGE plpgsql;
