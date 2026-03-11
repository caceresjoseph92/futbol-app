package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO users (id, name, email, password_hash, role) VALUES (gen_random_uuid(), $1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET password_hash = $3, role = $4`,
		"Admin", "admin@futbol.com", string(hash), "admin")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Admin user created: admin@futbol.com / admin123")
}
