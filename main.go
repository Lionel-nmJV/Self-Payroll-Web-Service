package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Struct untuk menyimpan informasi top up balance perusahaan
type TopUp struct {
	ID          int       `json:"id"`
	Amount      float64   `json:"amount"`
	Transaction time.Time `json:"transaction"`
}

// Struct untuk menyimpan informasi pengurangan balance perusahaan
type Deduction struct {
	ID          int       `json:"id"`
	Amount      float64   `json:"amount"`
	Transaction time.Time `json:"transaction"`
}

// Struct untuk menyimpan informasi jabatan
type Position struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
}

// Struct untuk menyimpan informasi employee
type Employee struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Position  Position  `json:"position"`
	SecretID  string    `json:"-"`
	Withdrawn bool      `json:"withdrawn"`
	LastMonth time.Time `json:"last_month"`
}

// Struct untuk menyimpan respons API
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var (
	db *sql.DB
)

// Handler untuk melakukan top up balance perusahaan
func TopUpBalanceHandler(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan nilai amount dari body request
	var topUp TopUp
	err := json.NewDecoder(r.Body).Decode(&topUp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// Simpan data top up ke database
	insertTopUp := `
		INSERT INTO top_up (amount, transaction)
		VALUES ($1, $2)
	`
	_, err = db.Exec(insertTopUp, topUp.Amount, time.Now())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to top up balance"})
		return
	}

	// Tambahkan balance perusahaan
	updateBalance := `
		UPDATE company
		SET balance = balance + $1
	`
	_, err = db.Exec(updateBalance, topUp.Amount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to update balance"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Balance topped up successfully"})
}

// Handler untuk melakukan pengurangan balance perusahaan
func DeductBalanceHandler(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan nilai amount dari body request
	var deduction Deduction
	err := json.NewDecoder(r.Body).Decode(&deduction)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// Simpan data pengurangan balance ke database
	insertDeduction := `
		INSERT INTO deduction (amount, transaction)
		VALUES ($1, $2)
	`
	_, err = db.Exec(insertDeduction, deduction.Amount, time.Now())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to deduct balance"})
		return
	}

	// Kurangi balance perusahaan
	updateBalance := `
		UPDATE company
		SET balance = balance - $1
	`
	_, err = db.Exec(updateBalance, deduction.Amount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to update balance"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Balance deducted successfully"})
}

// Handler untuk melakukan penarikan salary oleh employee
func WithdrawSalaryHandler(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan nilai employee ID dan secret ID dari query parameters
	employeeID := r.FormValue("employee_id")
	secretID := r.FormValue("secret_id")

	// Periksa apakah employee dengan ID yang diberikan ada
	queryEmployee := `
		SELECT e.id, e.name, e.secret_id, e.withdrawn, e.last_month, p.id, p.name, p.salary
		FROM employee AS e
		INNER JOIN position AS p ON e.position_id = p.id
		WHERE e.id = $1
	`
	row := db.QueryRow(queryEmployee, employeeID)
	var employee Employee
	err := row.Scan(
		&employee.ID,
		&employee.Name,
		&employee.SecretID,
		&employee.Withdrawn,
		&employee.LastMonth,
		&employee.Position.ID,
		&employee.Position.Name,
		&employee.Position.Salary,
	)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Employee not found"})
		return
	}

	// Periksa apakah secret ID yang diberikan cocok dengan secret ID employee
	if secretID != employee.SecretID {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Unauthorized"})
		return
	}

	// Periksa apakah employee sudah melakukan penarikan pada bulan ini
	currentMonth := time.Now().Month()
	if employee.LastMonth.Month() == currentMonth && employee.Withdrawn {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Salary already withdrawn this month"})
		return
	}

	// Kurangi balance perusahaan sesuai dengan besaran salary employee
	updateBalance := `
		UPDATE company
		SET balance = balance - $1
	`
	_, err = db.Exec(updateBalance, employee.Position.Salary)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to update balance"})
		return
	}

	// Update status penarikan salary employee
	updateEmployee := `
		UPDATE employee
		SET withdrawn = true, last_month = $1
		WHERE id = $2
	`
	_, err = db.Exec(updateEmployee, time.Now(), employee.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to update employee status"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Salary withdrawn successfully"})
}

// Handler untuk mengelola informasi jabatan
func ManagePositionHandler(w http.ResponseWriter, r *http.Request) {
	// Tambahkan kode untuk mengelola informasi jabatan
}

// Handler untuk mengelola informasi employee
func ManageEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	// Tambahkan kode untuk mengelola informasi employee
}

// Fungsi utama
func main() {
	// Koneksi ke database PostgreSQL
	connStr := "user=<postgres> password=<2804> dbname=<postgres> sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Membuat router menggunakan package gorilla/mux
	router := mux.NewRouter()

	// Mengatur route dan handler untuk setiap fitur
	router.HandleFunc("/topup", TopUpBalanceHandler).Methods("POST")
	router.HandleFunc("/deduct", DeductBalanceHandler).Methods("POST")
	router.HandleFunc("/withdraw", WithdrawSalaryHandler).Methods("POST")
	router.HandleFunc("/position", ManagePositionHandler).Methods("POST")
	router.HandleFunc("/employee", ManageEmployeeHandler).Methods("POST")

	// Menjalankan server pada port tertentu
	fmt.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
