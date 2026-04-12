ALTER TABLE auctions ADD COLUMN images JSON DEFAULT NULL AFTER image_url;

-- Migrate existing image_url to images JSON array
UPDATE auctions SET images = JSON_ARRAY(image_url) WHERE image_url IS NOT NULL AND image_url != '';

ALTER TABLE auctions DROP COLUMN image_url;
