package orm

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"larazeno-core/pkg/db"
	"larazeno-core/pkg/slots/sanctum"
	"golang.org/x/crypto/bcrypt"
)

// PluralizeModel converts Laravel Model name to table name (User -> users, Product -> products)
func PluralizeModel(modelName string) string {
	lower := strings.ToLower(modelName)
	if strings.HasSuffix(lower, "y") && !strings.HasSuffix(lower, "ay") && !strings.HasSuffix(lower, "ey") && !strings.HasSuffix(lower, "oy") {
		return strings.TrimSuffix(lower, "y") + "ies"
	}
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return lower + "es"
	}
	return lower + "s"
}

// Find retrieves a single record by primary key
func Find(modelName string, id interface{}) (map[string]interface{}, error) {
	table := PluralizeModel(modelName)
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", table)
	rows, err := db.DBConn.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMaps(rows)
	if err != nil || len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

// All retrieves all records for the given model
func All(modelName string) ([]map[string]interface{}, error) {
	table := PluralizeModel(modelName)
	query := fmt.Sprintf("SELECT * FROM %s", table)
	rows, err := db.DBConn.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsToMaps(rows)
}

// Create inserts a record dynamically into the model's table
func Create(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	table := PluralizeModel(modelName)

	now := time.Now().Format("2006-01-02 15:04:05")
	if _, ok := data["created_at"]; !ok {
		data["created_at"] = now
	}
	if _, ok := data["updated_at"]; !ok {
		data["updated_at"] = now
	}

	// Hash password automatically if inserting into users table
	if table == "users" {
		if rawPwd, ok := data["password"].(string); ok && !strings.HasPrefix(rawPwd, "$2y$") && !strings.HasPrefix(rawPwd, "$2a$") {
			hashed, err := bcrypt.GenerateFromPassword([]byte(rawPwd), bcrypt.DefaultCost)
			if err == nil {
				data["password"] = string(hashed)
			}
		}
	}

	var cols []string
	var placeholders []string
	var vals []interface{}

	for k, v := range data {
		cols = append(cols, k)
		placeholders = append(placeholders, "?")
		vals = append(vals, v)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	res, err := db.DBConn.DB.Exec(query, vals...)
	if err != nil {
		return nil, err
	}

	lastID, _ := res.LastInsertId()
	data["id"] = lastID
	return data, nil
}

// Update modifies an existing record by ID
func Update(modelName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	table := PluralizeModel(modelName)
	now := time.Now().Format("2006-01-02 15:04:05")
	data["updated_at"] = now

	delete(data, "id")

	var setClauses []string
	var vals []interface{}

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		vals = append(vals, v)
	}

	vals = append(vals, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, strings.Join(setClauses, ", "))
	_, err := db.DBConn.DB.Exec(query, vals...)
	if err != nil {
		return nil, err
	}

	return Find(modelName, id)
}

// Delete removes record by ID
func Delete(modelName string, id interface{}) error {
	table := PluralizeModel(modelName)
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", table)
	_, err := db.DBConn.DB.Exec(query, id)
	return err
}

// QueryBuilder supports chained Eloquent queries: Model::where(...)->orderBy(...)->limit(...)->get()
type QueryBuilder struct {
	ModelName   string
	WhereClause []string
	WhereArgs   []interface{}
	OrderByCol  string
	OrderDir    string
	LimitVal    int
	OffsetVal   int
}

func Query(modelName string) *QueryBuilder {
	return &QueryBuilder{
		ModelName: modelName,
	}
}

func (q *QueryBuilder) Where(col string, opOrVal interface{}, optionalVal ...interface{}) *QueryBuilder {
	if len(optionalVal) == 0 {
		q.WhereClause = append(q.WhereClause, fmt.Sprintf("%s = ?", col))
		q.WhereArgs = append(q.WhereArgs, opOrVal)
	} else {
		op := fmt.Sprintf("%v", opOrVal)
		val := optionalVal[0]
		q.WhereClause = append(q.WhereClause, fmt.Sprintf("%s %s ?", col, op))
		q.WhereArgs = append(q.WhereArgs, val)
	}
	return q
}

func (q *QueryBuilder) OrderBy(col string, dir ...string) *QueryBuilder {
	q.OrderByCol = col
	if len(dir) > 0 && strings.ToUpper(dir[0]) == "DESC" {
		q.OrderDir = "DESC"
	} else {
		q.OrderDir = "ASC"
	}
	return q
}

func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.LimitVal = limit
	return q
}

func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.OffsetVal = offset
	return q
}

func (q *QueryBuilder) Get() ([]map[string]interface{}, error) {
	table := PluralizeModel(q.ModelName)
	sqlStr := fmt.Sprintf("SELECT * FROM %s", table)

	if len(q.WhereClause) > 0 {
		sqlStr += " WHERE " + strings.Join(q.WhereClause, " AND ")
	}
	if q.OrderByCol != "" {
		sqlStr += fmt.Sprintf(" ORDER BY %s %s", q.OrderByCol, q.OrderDir)
	}
	if q.LimitVal > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", q.LimitVal)
	}
	if q.OffsetVal > 0 {
		sqlStr += fmt.Sprintf(" OFFSET %d", q.OffsetVal)
	}

	rows, err := db.DBConn.DB.Query(sqlStr, q.WhereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsToMaps(rows)
}

func (q *QueryBuilder) First() (map[string]interface{}, error) {
	q.Limit(1)
	results, err := q.Get()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

func (q *QueryBuilder) Count() (int, error) {
	table := PluralizeModel(q.ModelName)
	sqlStr := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	if len(q.WhereClause) > 0 {
		sqlStr += " WHERE " + strings.Join(q.WhereClause, " AND ")
	}

	var count int
	err := db.DBConn.DB.QueryRow(sqlStr, q.WhereArgs...).Scan(&count)
	return count, err
}

// FindByField finds the first record matching field=value
func FindByField(modelName string, field string, value interface{}) (map[string]interface{}, error) {
	return Query(modelName).Where(field, value).First()
}

// VerifyCredentials validates email/password and returns a Sanctum token
func VerifyCredentials(modelName string, email string, password string, tokenName string) (map[string]interface{}, string, error) {
	user, err := FindByField(modelName, "email", email)
	if err != nil {
		return nil, "", fmt.Errorf("user not found")
	}

	hashedPwd, ok := user["password"].(string)
	if !ok {
		return nil, "", fmt.Errorf("invalid password hash")
	}

	// Verify bcrypt password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("invalid password")
	}

	var userID int64 = 1
	if idVal, ok := user["id"]; ok {
		switch v := idVal.(type) {
		case int64:
			userID = v
		case int:
			userID = int64(v)
		case float64:
			userID = int64(v)
		}
	}

	tokenRecord, err := sanctum.CreateToken("App\\Models\\"+modelName, userID, tokenName, []string{"*"})
	if err != nil {
		return nil, "", err
	}

	// Remove password hash from returned user map
	delete(user, "password")

	return user, tokenRecord.PlainTextToken, nil
}

func scanRowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if val == nil || *val == nil {
				m[colName] = nil
				continue
			}
			b, ok := (*val).([]byte)
			if ok {
				m[colName] = string(b)
			} else {
				m[colName] = *val
			}
		}
		results = append(results, m)
	}
	return results, nil
}
