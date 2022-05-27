TRUNCATE "providers" RESTART IDENTITY CASCADE;
TRUNCATE "categories" RESTART IDENTITY CASCADE;
TRUNCATE "entries" RESTART IDENTITY CASCADE;
TRUNCATE "blocked_categories" RESTART IDENTITY CASCADE;
TRUNCATE "entry_to_categories" RESTART IDENTITY CASCADE;

INSERT INTO "providers" ("id", "url", "lang", "blocked_words")
    SELECT "id", "url", "lang", "blocked_words" FROM "old_providers";
INSERT INTO "categories" ("id", "name", "provider_id")
    SELECT "id", "name", "provider_id" FROM "old_categories";
INSERT INTO "entries" ("id", "link", "title", "description", "published_at", "message_id", "provider_id", "updated_at")
    SELECT "guid", "link", "title", "description", "published", "message_id", "provider_id", "updated_at" FROM "old_entries";
INSERT INTO "blocked_categories" ("category_id")
    SELECT "category_id" FROM "old_blocked_categories";
INSERT INTO "entry_to_categories" ("entry_id", "category_id")
    SELECT "entry_id", "category_id" FROM "old_entry_to_categories";