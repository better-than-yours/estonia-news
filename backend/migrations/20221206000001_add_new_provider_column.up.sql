ALTER TABLE "providers"
    ADD COLUMN "name" text;

UPDATE "providers" SET "name" = 'ERR' WHERE "url" = 'https://news.err.ee/rss';
UPDATE "providers" SET "name" = 'ERR' WHERE "url" = 'https://rus.err.ee/rss';
UPDATE "providers" SET "name" = 'ERR' WHERE "url" = 'https://www.err.ee/rss';
UPDATE "providers" SET "name" = 'Postimees' WHERE "url" = 'https://news.postimees.ee/rss';
UPDATE "providers" SET "name" = 'Postimees' WHERE "url" = 'https://rus.postimees.ee/rss';
UPDATE "providers" SET "name" = 'Postimees' WHERE "url" = 'https://www.postimees.ee/rss';
UPDATE "providers" SET "name" = 'Delfi' WHERE "url" = 'https://feeds.feedburner.com/rusdelfinews';
UPDATE "providers" SET "name" = 'Delfi' WHERE "url" = 'https://feeds.feedburner.com/delfiuudised';
