ALTER TABLE "entries"
    ADD COLUMN "image_url" text,
    ADD COLUMN "paywall" bool NOT NULL DEFAULT 'false';