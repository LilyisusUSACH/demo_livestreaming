package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Estructura de la base de datos PostgreSQL
type DB struct {
	*sql.DB
}

// Modelo de Usuario registrado en el sistema
type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

// Configuración de conexión a PostgreSQL
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// InitDB inicializa la conexión con la base de datos PostgreSQL y crea la tabla de usuarios
func InitDB(cfg Config) (*DB, error) {
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir conexión postgres: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("error al conectar con postgres en %s:%s: %w", cfg.Host, cfg.Port, err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := database.Exec(query); err != nil {
		return nil, fmt.Errorf("error al crear tabla de usuarios en postgres: %w", err)
	}

	return &DB{database}, nil
}

// CreateUser registra un nuevo usuario en PostgreSQL
func (db *DB) CreateUser(id, name, email, passwordHash string) (*User, error) {
	query := `INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(query, id, name, email, passwordHash)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        id,
		Name:      name,
		Email:     email,
		CreatedAt: "",
	}, nil
}

// GetUserByEmail busca un usuario por correo electrónico
func (db *DB) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE email = $1`
	row := db.QueryRow(query, email)

	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetUserByID busca un usuario por su identificador único UUID
func (db *DB) GetUserByID(id string) (*User, error) {
	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE id = $1`
	row := db.QueryRow(query, id)

	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// UserExists verifica si un correo electrónico ya existe en la base de datos
func (db *DB) UserExists(email string) (bool, error) {
	u, err := db.GetUserByEmail(email)
	if err != nil || u == nil {
		return false, nil
	}
	return true, nil
}
