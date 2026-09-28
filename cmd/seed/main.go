// Seed importer: loads normalized catalog JSON (produced by the customer app's
// tool/dump_catalog.dart) into Postgres — restaurants, categories (food+grocery
// hierarchy) and food_products. Idempotent: deterministic IDs + ON CONFLICT.
//
//	go run ./cmd/seed -file /tmp/catalog.json
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"dailzo/config"
	"dailzo/db"
)

type catalog struct {
	Restaurants []restaurant `json:"restaurants"`
	Grocery     []groCat     `json:"grocery"`
}
type restaurant struct {
	Name         string     `json:"name"`
	Image        string     `json:"image"`
	Rating       string     `json:"rating"`
	DeliveryTime string     `json:"delivery_time"`
	Cuisines     string     `json:"cuisines"`
	Location     string     `json:"location"`
	Items        []foodItem `json:"items"`
}
type foodItem struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Price       json.Number `json:"price"`
	ActualPrice json.Number `json:"actual_price"`
	Image       string      `json:"image"`
	Category    string      `json:"category"`
	Veg         bool        `json:"veg"`
	PrepTime    json.Number `json:"prep_time"`
}
type groCat struct {
	Category   string   `json:"category"`
	Image      string   `json:"image"`
	Department string   `json:"department"`
	Subs       []groSub `json:"subcategories"`
}
type groSub struct {
	Name  string    `json:"name"`
	Image string    `json:"image"`
	Items []groItem `json:"items"`
}
type groItem struct {
	Name        string      `json:"name"`
	Unit        string      `json:"unit"`
	Quantity    string      `json:"quantity"`
	Price       json.Number `json:"price"`
	ActualPrice json.Number `json:"actual_price"`
	Image       string      `json:"image"`
	Brand       string      `json:"brand"`
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 38 {
		s = s[:38]
	}
	return s
}

// productID returns a deterministic 34-char id (fits char(36)) for idempotency.
func productID(prefix, seed string) string {
	sum := md5.Sum([]byte(seed))
	return prefix + hex.EncodeToString(sum[:]) // 2 + 32 = 34 chars
}

// nullIfEmpty lets an empty string bind as SQL NULL instead of "" so
// department stays genuinely unset for rows that don't have one.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func num(n json.Number, def float64) float64 {
	if n == "" {
		return def
	}
	f, err := n.Float64()
	if err != nil {
		return def
	}
	return f
}

func numStr(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

func main() {
	file := flag.String("file", "/tmp/catalog.json", "path to catalog JSON")
	flag.Parse()

	raw, err := os.ReadFile(*file)
	if err != nil {
		fmt.Println("read catalog:", err)
		os.Exit(1)
	}
	var cat catalog
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&cat); err != nil {
		fmt.Println("parse catalog:", err)
		os.Exit(1)
	}

	cfg := config.LoadConfig()
	db.ConnectDatabase(cfg)
	defer db.CloseDatabase()
	ctx := context.Background()
	pool := db.DB

	var restCount, foodCount, catCount, groCount int

	// ─── FOOD ───────────────────────────────────────────────────────────
	foodCats := map[string]bool{} // category name -> seeded
	for _, r := range cat.Restaurants {
		rid := "RST_" + slug(r.Name)
		addr := r.Location
		if strings.TrimSpace(addr) == "" {
			addr = "Mumbai, Maharashtra"
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO restaurants (id, name, address, image_path, cuisines, delivery_time, location, created_on, last_updated_on)
			VALUES ($1,$2,$3,$4,$5,$6,$7,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, address=EXCLUDED.address, image_path=EXCLUDED.image_path,
				cuisines=EXCLUDED.cuisines, delivery_time=EXCLUDED.delivery_time, location=EXCLUDED.location`,
			rid, r.Name, addr, r.Image, r.Cuisines, r.DeliveryTime, r.Location)
		if err != nil {
			fmt.Println("restaurant:", r.Name, err)
			continue
		}
		restCount++

		for _, it := range r.Items {
			catID := "CATF_" + slug(it.Category)
			if !foodCats[catID] {
				_, err = pool.Exec(ctx, `
					INSERT INTO categories (id, name, kind, sort_order, created_on, last_updated_on)
					VALUES ($1,$2,'food',0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
					ON CONFLICT (id) DO NOTHING`, catID, it.Category)
				if err == nil {
					foodCats[catID] = true
					catCount++
				}
			}
			ptype := "non_veg"
			if it.Veg {
				ptype = "veg"
			}
			pid := productID("PF", r.Name+"|"+it.Name)
			_, err = pool.Exec(ctx, `
				INSERT INTO food_products (id, name, description, category, category_id, kind, type,
					price, image_url, is_active, stock_quantity, restaurant_id, created_on, last_updated_on)
				VALUES ($1,$2,$3,$4,$5,'food',$6,$7,$8,true,50,$9,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
				ON CONFLICT (id) DO UPDATE SET price=EXCLUDED.price, image_url=EXCLUDED.image_url,
					category_id=EXCLUDED.category_id, restaurant_id=EXCLUDED.restaurant_id, is_active=true`,
				pid, it.Name, it.Description, it.Category, catID, ptype,
				num(it.Price, 0), it.Image, rid)
			if err != nil {
				fmt.Println("food product:", it.Name, err)
				continue
			}
			foodCount++
		}
	}

	// ─── GROCERY ────────────────────────────────────────────────────────
	for _, c := range cat.Grocery {
		parentID := "CATG_" + slug(c.Category)
		_, err := pool.Exec(ctx, `
			INSERT INTO categories (id, name, kind, image_url, department, sort_order, created_on, last_updated_on)
			VALUES ($1,$2,'grocery',$3,$4,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, image_url=EXCLUDED.image_url,
				department=EXCLUDED.department`,
			parentID, c.Category, c.Image, nullIfEmpty(c.Department))
		if err != nil {
			fmt.Println("grocery cat:", c.Category, err)
			continue
		}
		catCount++
		for _, s := range c.Subs {
			subID := "CATGS_" + slug(c.Category+"_"+s.Name)
			_, err = pool.Exec(ctx, `
				INSERT INTO categories (id, name, kind, parent_id, image_url, sort_order, created_on, last_updated_on)
				VALUES ($1,$2,'grocery',$3,$4,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
				ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, parent_id=EXCLUDED.parent_id, image_url=EXCLUDED.image_url`,
				subID, s.Name, parentID, s.Image)
			if err != nil {
				fmt.Println("grocery subcat:", s.Name, err)
				continue
			}
			catCount++
			for _, it := range s.Items {
				desc := strings.TrimSpace(it.Quantity + " " + it.Unit)
				pid := productID("PG", c.Category+"|"+s.Name+"|"+it.Name)
				_, err = pool.Exec(ctx, `
					INSERT INTO food_products (id, name, description, category, category_id, kind, type,
						price, image_url, is_active, stock_quantity, unit, weight, brand, created_on, last_updated_on)
					VALUES ($1,$2,$3,$4,$5,'grocery','veg',$6,$7,true,100,$8,$9,$10,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
					ON CONFLICT (id) DO UPDATE SET price=EXCLUDED.price, image_url=EXCLUDED.image_url,
						category_id=EXCLUDED.category_id, is_active=true, brand=EXCLUDED.brand`,
					pid, it.Name, desc, s.Name, subID, num(it.Price, 0), it.Image, it.Unit, numStr(it.Quantity), it.Brand)
				if err != nil {
					fmt.Println("grocery product:", it.Name, err)
					continue
				}
				groCount++
			}
		}
	}

	fmt.Printf("Seed complete: restaurants=%d categories=%d food_products=%d grocery_products=%d\n",
		restCount, catCount, foodCount, groCount)
}
