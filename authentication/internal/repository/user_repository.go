package repository

import (
	"context"
	"fmt"
	db "job_portal/authentication/db/sqlc" // SQLC generated code for interacting with the database
	token "job_portal/authentication/interfaces/middleware/token_maker"
	"job_portal/authentication/internal/domain/entity"
	"job_portal/authentication/pkg/utils"
	"log"
	"time"
)

// UserRepository implements the AuthRepository interface.
// This struct interacts with the database using SQLC-generated code.
type UserRepository struct {
	store db.Store
}

// CreateToken implements repository.UserRepository.
func (r *UserRepository) CreateToken(ctx context.Context, email string) (string, error) {
	// // Load configuration
	// configs, err := config.LoadConfig("...")
	// if err != nil {
	// 	log.Fatalf("Failed to load env file: %v", err)
	// }

	tokenMaker, err := token.NewTokenMaker("beb4118e1bdc8020df695ceec7e464a5")
	if err != nil {
		log.Fatalf("Failed to load env file")

	}

	duration := time.Hour * 24
	accessToken, _, err := tokenMaker.CreateToken(email, duration)
	if err != nil {
		return "", fmt.Errorf("some went wrong: %d", err)
	}

	return accessToken, nil
}

// NewUserRepository creates a new instance of UserRepository.
func NewUserRepository(store db.Store) *UserRepository {
	return &UserRepository{
		store: store,
	}
}

// GetUserByEmail retrieves a user by their email from the database.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {

	userDetails, err := r.store.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	return &entity.User{
		FullName:  userDetails.FullName,
		Email:     userDetails.Email,
		CreatedAt: userDetails.CreatedAt,
		Password:  userDetails.HashedPassword,
	}, nil
}

// CreateUser creates a new user in the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return err
	}

	_, err = r.store.CreateUser(ctx, db.CreateUserParams{
		Email:          user.Email,
		FullName:       user.FullName,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		return err
	}

	return nil
}

// UpdatePassword updates a user's password in the database.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, newPassword string) error {

	_, err := r.store.UpdateUser(ctx, db.UpdateUserParams{

		HashedPassword: newPassword,
	})

	return err
}
