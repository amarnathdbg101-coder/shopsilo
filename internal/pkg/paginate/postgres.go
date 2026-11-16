package paginate

import (
	"fmt"
	"strings"
)

// SQLClause generates a parameterized PostgreSQL ORDER BY, LIMIT, and OFFSET clause.
// startArgIndex specifies the positional index for $1, $2... so you can append to existing WHERE parameters.
//
// Example:
//
//	clause, args := paginate.SQLClause(params, 1)
//	// Returns: "ORDER BY \"created_at\" DESC LIMIT $1 OFFSET $2", []any{10, 0}
func SQLClause(p Params, startArgIndex int) (string, []any) {
	if startArgIndex <= 0 {
		startArgIndex = 1
	}

	quotedCol := quoteIdentifier(p.Sort)
	order := OrderDesc
	if strings.EqualFold(p.Order, OrderAsc) {
		order = OrderAsc
	}

	limitPlaceholder := fmt.Sprintf("$%d", startArgIndex)
	offsetPlaceholder := fmt.Sprintf("$%d", startArgIndex+1)

	clause := fmt.Sprintf("ORDER BY %s %s LIMIT %s OFFSET %s",
		quotedCol,
		order,
		limitPlaceholder,
		offsetPlaceholder,
	)

	args := []any{p.Limit(), p.Offset()}
	return clause, args
}

// BuildCountSQL wraps any valid SQL query in a count subquery.
// GROUP BY, DISTINCT, aur complex JOIN queries me standard SELECT COUNT(*) directly replace karne se fail hota hai.
// Subquery wrapper har case me 100% accurate count return karta hai.
//
// Example:
//
//	countSQL := paginate.BuildCountSQL("SELECT p.id, c.name FROM products p JOIN categories c ON ... WHERE p.price > $1")
//	// Returns: "SELECT COUNT(*) FROM (SELECT p.id, c.name FROM products p JOIN categories c ON ... WHERE p.price > $1) AS _count_subquery"
func BuildCountSQL(baseQuery string) string {
	trimmed := strings.TrimSpace(baseQuery)
	// Remove trailing semicolon if present
	trimmed = strings.TrimSuffix(trimmed, ";")

	return fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS _count_subquery", trimmed)
}

// CursorWhereClause generates parameterized PostgreSQL condition and ORDER BY for keyset cursor pagination.
// If order is DESC, condition is: "column < $N"
// If order is ASC, condition is: "column > $N"
func CursorWhereClause(column, cursorVal, order string, startArgIndex int) (string, string, []any) {
	if startArgIndex <= 0 {
		startArgIndex = 1
	}

	quotedCol := quoteIdentifier(column)
	cleanOrder := OrderDesc
	operator := "<"

	if strings.EqualFold(order, OrderAsc) {
		cleanOrder = OrderAsc
		operator = ">"
	}

	orderClause := fmt.Sprintf("ORDER BY %s %s", quotedCol, cleanOrder)

	if cursorVal == "" {
		return "", orderClause, nil
	}

	whereClause := fmt.Sprintf("%s %s $%d", quotedCol, operator, startArgIndex)
	return whereClause, orderClause, []any{cursorVal}
}

// quoteIdentifier safely quotes SQL identifiers (e.g. "users"."created_at" or "id")
// to prevent keyword collisions (like user, order, group) in PostgreSQL.
func quoteIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return `"id"`
	}

	parts := strings.Split(s, ".")
	var quoted []string
	for _, p := range parts {
		cleaned := cleanIdentifier(p)
		if cleaned != "" {
			quoted = append(quoted, fmt.Sprintf(`"%s"`, cleaned))
		}
	}

	if len(quoted) == 0 {
		return `"id"`
	}
	return strings.Join(quoted, ".")
}
