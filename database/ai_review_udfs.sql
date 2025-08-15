-- 文件：database/ai_review_udfs.sql

-- 1. 初始化函数
CREATE OR REPLACE FUNCTION ai_review_init()
RETURNS VOID AS $$
BEGIN
    -- 安装必要扩展
    CREATE EXTENSION IF NOT EXISTS plpython3u;
    CREATE EXTENSION IF NOT EXISTS pgvector;
END;
$$ LANGUAGE plpgsql;

-- 2. 主要分析函数
CREATE OR REPLACE FUNCTION ai_review_analyze(
    review_content TEXT,
    user_rating INTEGER
) RETURNS JSONB AS $$
DECLARE
    raw_result TEXT;
    cleaned_result TEXT;
    result JSONB;
BEGIN
    -- 调用DeepSeek API
    SELECT openai.prompt(
        -- System Prompt: 定义AI的角色和任务
        '你是一个专业的外卖平台评价分析师。你的任务是基于用户评价和评分，返回一个结构化的JSON对象。JSON必须包含以下字段：emotional_intensity (情感强度, 1-5), delivery_issue (是否配送问题, bool), food_quality_issue (是否食物质量问题, bool), service_issue (是否服务问题, bool), summary_20chars (20字摘要), suggested_reply (给商家的回复建议)。',
        -- User Prompt: 用户的实际评价内容
        '评价内容：' || review_content || ' | 用户评分：' || user_rating || '/5'
    ) INTO raw_result;

    -- 移除Markdown代码块标记
    cleaned_result := regexp_replace(raw_result, '```json\n|\n```', '', 'g');

    -- 将清理后的字符串转换为JSONB
    result := cleaned_result::jsonb;

    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- 3. 清理函数
CREATE OR REPLACE FUNCTION ai_review_deinit()
RETURNS VOID AS $$
BEGIN
    -- 清理任务
    TRUNCATE some_temporary_table;
END;
$$ LANGUAGE plpgsql;