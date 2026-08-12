package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type MenuItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       int     `json:"price"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
}

type OrderItem struct {
	ID         int    `json:"id"`
	OrderID    int    `json:"order_id"`
	MenuItemID int    `json:"menu_item_id"`
	Name       string `json:"name"`
	Price      int    `json:"price"`
	Quantity   int    `json:"quantity"`
	Spiciness  string `json:"spiciness"`
}

type Order struct {
	ID        int         `json:"id"`
	TableNo   string      `json:"table_no"`
	Status    string      `json:"status"` // "ordered", "paid"
	Total     int         `json:"total"`
	CreatedAt string      `json:"created_at"`
	Items     []OrderItem `json:"items"`
}

type CreateOrderRequest struct {
	TableNo string `json:"table_no"`
	Items   []struct {
		MenuItemID int    `json:"menu_item_id"`
		Quantity   int    `json:"quantity"`
		Spiciness  string `json:"spiciness"`
	} `json:"items"`
}

type CheckoutRequest struct {
	TableNo string `json:"table_no"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./curry.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/menu", handleMenu)
	mux.HandleFunc("/api/orders", handleOrders)
	mux.HandleFunc("/api/orders/history", handleOrderHistory)
	mux.HandleFunc("/api/checkout", handleCheckout)

	corsMux := corsMiddleware(mux)

	log.Println("ネパールカレー オーダーシステム サーバー起動中 (:8080)...")
	if err := http.ListenAndServe(":8080", corsMux); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initDB() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS menu_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			category TEXT,
			price INTEGER,
			description TEXT,
			image_url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_no TEXT,
			status TEXT,
			total INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER,
			menu_item_id INTEGER,
			quantity INTEGER,
			spiciness TEXT,
			FOREIGN KEY(order_id) REFERENCES orders(id)
		);`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatal(err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM menu_items").Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO menu_items (name, category, price, description, image_url) VALUES 
			('チキンカレー', 'Curry', 900, 'スパイスの効いた王道のネパール風チキンカレー', 'https://images.unsplash.com/photo-1588166524941-3bf61a9c41db?w=500'),
			('マトンカレー', 'Curry', 1050, 'じっくり煮込んだ羊肉のコク深いスパイスカレー', 'https://images.unsplash.com/photo-1545247181-516773cae754?w=500'),
			('ダルバート (Dal Bhat)', 'Set', 1200, 'ネパールの定食。豆スープ、カレー、タルカリの完璧なセット', 'https://images.unsplash.com/photo-1626777552726-4a6b54c97e46?w=500'),
			('プレーンナン', 'Bread', 300, 'タンドール窯で焼き上げたモチモチのナン', 'https://images.unsplash.com/photo-1601050690597-df0568f70950?w=500'),
			('チーズナン', 'Bread', 500, 'とろーりチーズがたっぷり詰まった人気のナン', 'https://images.unsplash.com/photo-1626074353765-517a681e40be?w=500'),
			('モモ (Momo 6個)', 'Side', 600, 'スパイシーな肉汁溢れるネパール風蒸し餃子', 'https://images.unsplash.com/photo-1534422298391-e4f8c172dddb?w=500'),
			('マンゴーラッシー', 'Drink', 400, 'カレーにぴったりの濃厚で甘いマンゴーヨーグルトドリンク', 'https://images.unsplash.com/photo-1528823872057-9c018a7a70b3?w=500')`)
	}
}

func handleMenu(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, category, price, description, image_url FROM menu_items")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []MenuItem
	for rows.Next() {
		var item MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.Description, &item.ImageURL); err == nil {
			items = append(items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "注文アイテムが空です", http.StatusBadRequest)
		return
	}

	total := 0
	type orderDetail struct {
		menuItemID int
		quantity   int
		spiciness  string
		price      int
	}
	var details []orderDetail

	for _, item := range req.Items {
		var price int
		err := db.QueryRow("SELECT price FROM menu_items WHERE id = ?", item.MenuItemID).Scan(&price)
		if err != nil {
			continue
		}
		total += price * item.Quantity
		details = append(details, orderDetail{
			menuItemID: item.MenuItemID,
			quantity:   item.Quantity,
			spiciness:  item.Spiciness,
			price:      price,
		})
	}

	res, err := db.Exec("INSERT INTO orders (table_no, status, total) VALUES (?, 'ordered', ?)", req.TableNo, total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderID, _ := res.LastInsertId()

	for _, d := range details {
		_, _ = db.Exec("INSERT INTO order_items (order_id, menu_item_id, quantity, spiciness) VALUES (?, ?, ?, ?)",
			orderID, d.menuItemID, d.quantity, d.spiciness)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "注文が完了しました",
		"order_id": orderID,
	})
}

func handleOrderHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tableNo := r.URL.Query().Get("table_no")
	if tableNo == "" {
		tableNo = "1"
	}

	rows, err := db.Query("SELECT id, table_no, status, total, created_at FROM orders WHERE table_no = ? AND status = 'ordered' ORDER BY id DESC", tableNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.TableNo, &o.Status, &o.Total, &o.CreatedAt); err != nil {
			continue
		}

		itemRows, err := db.Query(`
			SELECT oi.id, oi.order_id, oi.menu_item_id, m.name, m.price, oi.quantity, oi.spiciness 
			FROM order_items oi 
			JOIN menu_items m ON oi.menu_item_id = m.id 
			WHERE oi.order_id = ?`, o.ID)

		if err == nil {
			o.Items = []OrderItem{}
			for itemRows.Next() {
				var item OrderItem
				if err := itemRows.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Name, &item.Price, &item.Quantity, &item.Spiciness); err == nil {
					o.Items = append(o.Items, item)
				}
			}
			itemRows.Close()
		}

		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var grandTotal int
	err := db.QueryRow("SELECT COALESCE(SUM(total), 0) FROM orders WHERE table_no = ? AND status = 'ordered'", req.TableNo).Scan(&grandTotal)
	if err != nil || grandTotal == 0 {
		http.Error(w, "精算対象の注文がありません", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE orders SET status = 'paid' WHERE table_no = ? AND status = 'ordered'", req.TableNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "精算が完了しました。ご来店ありがとうございました！",
		"grand_total": grandTotal,
	})
}