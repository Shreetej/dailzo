# Post-seed fixups (customer catalog)

Run these AFTER `go run ./cmd/seed` to reconcile the customer app's bundled
image assets and to populate product brands. They are idempotent and match rows
by content-stable ids / name patterns.

    psql -d dailzo -f scripts/post_seed_01_image_remap.sql
    psql -d dailzo -f scripts/post_seed_02_brands.sql

- **01_image_remap**: repoints restaurant/product image paths that used
  non-bundled filenames (e.g. `temptation_chinese.png`) to the actual bundled
  asset (`temptationchinese.jpg`). Only ~88 of ~1059 catalog images exist in the
  app bundle; the remainder are a genuine asset gap (product images were never
  added to `assets/images/`) and render as graceful placeholders via AppImage.
- **02_brands**: tags grocery products with a `brand` (Coca-Cola, Amul, Lay's,
  Haldiram, Nestle, Cadbury, Britannia, P&G) by keyword, so the customer app's
  "Most Loved Brands" / brand pages derive from real server data.
