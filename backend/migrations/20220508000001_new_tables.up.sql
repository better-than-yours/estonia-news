CREATE SEQUENCE IF NOT EXISTS providers_id_seq;
CREATE TABLE "providers" (
    "id" int8 NOT NULL DEFAULT nextval('providers_id_seq'::regclass),
    "url" text,
    "lang" text,
    "blocked_words" _text,
    PRIMARY KEY ("id")
);

CREATE SEQUENCE IF NOT EXISTS categories_id_seq;
CREATE TABLE "categories" (
    "id" int8 NOT NULL DEFAULT nextval('categories_id_seq'::regclass),
    "name" text NOT NULL,
    "provider_id" int8 NOT NULL,
    CONSTRAINT "fk_categories_provider" FOREIGN KEY ("provider_id") REFERENCES "providers"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uniq_idx_categories" ON "categories"("name", "provider_id");

CREATE TABLE "entries" (
    "id" text NOT NULL,
    "link" text,
    "title" text,
    "description" text,
    "published_at" timestamptz NOT NULL,
    "message_id" int8,
    "provider_id" int8 NOT NULL,
    "updated_at" timestamptz NOT NULL,
    CONSTRAINT "fk_entries_provider" FOREIGN KEY ("provider_id") REFERENCES "providers"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

CREATE SEQUENCE IF NOT EXISTS blocked_categories_category_id_seq;
CREATE TABLE "blocked_categories" (
    "category_id" int8 NOT NULL DEFAULT nextval('blocked_categories_category_id_seq'::regclass),
    CONSTRAINT "fk_blocked_categories_category" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("category_id")
);

CREATE TABLE "entry_to_categories" (
    "entry_id" text NOT NULL,
    "category_id" int8 NOT NULL,
    CONSTRAINT "fk_entry_to_categories_entry" FOREIGN KEY ("entry_id") REFERENCES "entries"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "fk_entry_to_categories_category" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("entry_id","category_id")
);