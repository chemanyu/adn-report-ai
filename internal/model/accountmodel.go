package model

import (
	"context"
	"database/sql"
	"errors"
)

type AccountModel interface {
	List(context.Context) ([]Account, error)
	Insert(context.Context, string, string, int64) (Account, error)
	Exists(context.Context, int64) (bool, error)
}

type accountModel struct{ db *sql.DB }

func NewAccountModel(db *sql.DB) AccountModel { return &accountModel{db: db} }
func (m *accountModel) List(ctx context.Context) ([]Account, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT id,name,account_code FROM adn_accounts ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Account{}
	for rows.Next() {
		var a Account
		if err = rows.Scan(&a.ID, &a.Name, &a.Code); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
func (m *accountModel) Insert(ctx context.Context, name, code string, userID int64) (Account, error) {
	a := Account{Name: name, Code: code}
	err := m.db.QueryRowContext(ctx, "INSERT INTO adn_accounts(name,account_code,created_by) VALUES($1,$2,$3) RETURNING id", name, code, userID).Scan(&a.ID)
	return a, err
}
func (m *accountModel) Exists(ctx context.Context, id int64) (bool, error) {
	var found int
	err := m.db.QueryRowContext(ctx, "SELECT 1 FROM adn_accounts WHERE id=$1", id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
