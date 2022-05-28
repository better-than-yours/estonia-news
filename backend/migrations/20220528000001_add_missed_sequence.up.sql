CREATE SEQUENCE IF NOT EXISTS providers_id_seq;
ALTER TABLE "providers" ALTER COLUMN "id" SET DEFAULT nextval('providers_id_seq'::regclass);

CREATE SEQUENCE IF NOT EXISTS categories_id_seq;
ALTER TABLE "categories" ALTER COLUMN "id" SET DEFAULT nextval('categories_id_seq'::regclass);

CREATE SEQUENCE IF NOT EXISTS blocked_categories_category_id_seq;
ALTER TABLE "blocked_categories" ALTER COLUMN "category_id" SET DEFAULT nextval('blocked_categories_category_id_seq'::regclass);