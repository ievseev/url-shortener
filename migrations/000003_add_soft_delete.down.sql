ALTER TABLE user_urls
    DROP COLUMN IF EXISTS is_creator;

ALTER TABLE short_urls
    DROP COLUMN IF EXISTS is_deleted;
