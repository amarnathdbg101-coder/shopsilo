package repository

import (
	"context"
	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewUserRepo(db *pgxpool.Pool, logger *zap.Logger) *UserRepo {
	return &UserRepo{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepo) Register(ctx context.Context, user model.User) error {
	query := `insert into users (name,email,password) values ($1,$2,$3)`
	_, err := r.db.Exec(ctx, query, user.Name, user.Email, user.Password)
	return err
}

func(r *UserRepo) FindByEmail(ctx context.Context,email string) (*model.User,error) {
	var user model.User
	query := `select id,name,password from users where email=$1`
	err := r.db.QueryRow(ctx,query,email).Scan(&user.Id,&user.Name,&user.Password)
	if err != nil {
		return nil,err
	}
	return &user,nil
}
