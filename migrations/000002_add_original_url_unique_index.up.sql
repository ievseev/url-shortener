CREATE UNIQUE INDEX IF NOT EXISTS short_urls_original_url_uidx
ON short_urls (original_url);
