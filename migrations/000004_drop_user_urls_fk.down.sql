ALTER TABLE user_urls
    ADD CONSTRAINT user_urls_url_id_fkey
        FOREIGN KEY (url_id) REFERENCES short_urls(id) ON DELETE CASCADE;
