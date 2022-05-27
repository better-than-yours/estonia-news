CREATE SEQUENCE IF NOT EXISTS blocked_categories_category_id_seq;
CREATE TABLE IF NOT EXISTS "blocked_categories" (
    "category_id" int8 NOT NULL DEFAULT nextval('blocked_categories_category_id_seq'::regclass),
    PRIMARY KEY ("category_id")
);

CREATE SEQUENCE IF NOT EXISTS categories_id_seq;
CREATE TABLE IF NOT EXISTS "categories" (
    "id" int8 NOT NULL DEFAULT nextval('categories_id_seq'::regclass),
    "name" text,
    "provider_id" int8,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "entries" (
    "guid" text,
    "link" text,
    "title" text,
    "description" text,
    "published" timestamptz,
    "message_id" int8,
    "provider_id" int8,
    "updated_at" timestamptz
);

CREATE TABLE IF NOT EXISTS "entry_to_categories" (
    "entry_id" text NOT NULL,
    "category_id" int8 NOT NULL,
    PRIMARY KEY ("entry_id","category_id")
);

CREATE SEQUENCE IF NOT EXISTS providers_id_seq;
CREATE TABLE IF NOT EXISTS "providers" (
    "id" int8 NOT NULL DEFAULT nextval('providers_id_seq'::regclass),
    "url" text,
    "lang" text,
    "blocked_words" _text,
    PRIMARY KEY ("id")
);

ALTER TABLE IF EXISTS "blocked_categories" RENAME TO "old_blocked_categories";
ALTER TABLE IF EXISTS "categories" RENAME TO "old_categories";
ALTER TABLE IF EXISTS "entries" RENAME TO "old_entries";
ALTER TABLE IF EXISTS "entry_to_categories" RENAME TO "old_entry_to_categories";
ALTER TABLE IF EXISTS "providers" RENAME TO "old_providers";