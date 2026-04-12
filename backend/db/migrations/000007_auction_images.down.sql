ALTER TABLE auctions ADD COLUMN image_url TEXT AFTER description;

UPDATE auctions SET image_url = JSON_UNQUOTE(JSON_EXTRACT(images, '$[0]')) WHERE images IS NOT NULL;

ALTER TABLE auctions DROP COLUMN images;
