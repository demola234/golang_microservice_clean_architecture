package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	token "github.com/demola234/api_gateway/infrastructure/middleware/token_maker"
	"github.com/demola234/authentication/config"
	db "github.com/demola234/authentication/db/sqlc" // SQLC generated code for interacting with the database
	"github.com/demola234/authentication/internal/domain/entity"
	"github.com/demola234/authentication/pkg/utils"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// UserRepository implements the AuthRepository interface.
type UserRepository struct {
	store db.Store
}

// UpdateUser implements repository.UserRepository.
func (r *UserRepository) UploadProfileImage(ctx context.Context, content io.Reader, userId uuid.UUID) (string, error) {
	// Create a new Cloudinary client
	configs, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("Could not load config", err)
	}

	// Upload the image to the cloudinary
	cld, _ := cloudinary.NewFromURL(configs.CloudinaryUrl)

	// Get the preferred name of the file if its not supplied
	result, err := cld.Upload.Upload(ctx, content, uploader.UploadParams{
		PublicID: userId.String(),
		Tags:     strings.Split(",", userId.String()),
	})

	if err != nil {
		return "", err
	}

	updateUser := db.UpdateUserProfilePictureParams{
		ID: userId,
		ProfilePicture: sql.NullString{
			String: result.SecureURL,
			Valid:  true,
		},
	}

	_, err = r.store.UpdateUserProfilePicture(ctx, updateUser)
	if err != nil {
		return "", err
	}

	return updateUser.ProfilePicture.String, nil

}

// UpdateUser implements repository.UserRepository.
func (r *UserRepository) UpdateUser(ctx context.Context, user *entity.User) error {

	// Check if the user exists
	updateUser := db.UpdateUserParams{
		ID:       user.ID,
		Name:     user.FullName,
		Username: user.Username,
		ProfilePicture: sql.NullString{
			String: user.ProfilePicture,
			Valid:  true,
		},
		Bio: sql.NullString{
			String: user.Bio,
			Valid:  true,
		},
		Phone: sql.NullString{
			String: user.Phone,
			Valid:  true,
		},
		Role: sql.NullString{
			String: user.Role,
			Valid:  true,
		},
		Email: user.Email,
		Password: sql.NullString{
			String: user.Password,
			Valid:  true,
		},
	}

	updatedUser, err := r.store.UpdateUser(ctx, updateUser)
	if err != nil {
		return err
	}

	// Update the user in the cache
	user.ID = updatedUser.ID
	user.FullName = updatedUser.Name
	user.Username = updatedUser.Username
	user.ProfilePicture = updatedUser.ProfilePicture.String
	user.IsActive = updatedUser.IsActive.Bool
	user.EmailVerified = updatedUser.EmailVerified.Bool
	user.Phone = updatedUser.Phone.String
	user.Provider = utils.ProviderType{
		Name:      updatedUser.Provider.String,
		ID:        updatedUser.ProviderID.String,
		TokenData: updatedUser.ProviderID.String,
	}
	user.EmailVerified = updatedUser.EmailVerified.Bool

	return nil

}

// CreateToken implements repository.UserRepository.
func (r *UserRepository) CreateToken(ctx context.Context, email string, userID string) (string, error) {
	// Load configuration
	configs, err := config.LoadConfig("../../")
	if err != nil {
		log.Fatalf("Failed to load env file: %v", err)
	}

	tokenMaker, err := token.NewTokenMaker(configs.TokenSymmetricKey)

	if err != nil {
		log.Fatalf("Failed to load env file: %s", err)
	}

	duration := time.Hour * 24
	accessToken, _, err := tokenMaker.CreateToken(email, userID, duration)
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
		return nil, fmt.Errorf("failed to retrieve user by email %s: %w", email, err)
	}

	password := userDetails.Password.String

	// Create ProviderType using the new unified format
	provider := utils.ProviderType{}
	if userDetails.Provider.Valid {
		provider.SetProvider(userDetails.Provider.String, userDetails.ProviderID.String)
	} else {
		// Default to local if provider not set
		provider.SetProvider("local", userDetails.Email)
	}

	return &entity.User{
		ID:             userDetails.ID,
		FullName:       userDetails.Name,
		Username:       userDetails.Username,
		Email:          userDetails.Email,
		Provider:       provider,
		ProviderID:     userDetails.ProviderID.String,
		ProfilePicture: userDetails.ProfilePicture.String,
		Role:           userDetails.Role.String,
		Password:       password,
		Phone:          userDetails.Phone.String,
		EmailVerified:  userDetails.EmailVerified.Bool,
		IsActive:       userDetails.IsActive.Bool,
		LastLogin:      userDetails.LastLogin.Time,
		CreatedAt:      userDetails.CreatedAt.Time,
		UpdatedAt:      userDetails.UpdatedAt.Time,
	}, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	// Get provider information using the unified method
	provider, providerID := user.Provider.GetProviderInfo()

	// Convert to sql.NullString
	var providerIDNullable sql.NullString
	if providerID != "" {
		providerIDNullable = sql.NullString{String: providerID, Valid: true}
	} else {
		providerIDNullable = sql.NullString{Valid: false}
	}

	// Hash password only for local users
	var hashedPasswordString sql.NullString
	if provider == "local" && user.Password != "" {
		hashedPassword, err := utils.HashPassword(user.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		hashedPasswordString = sql.NullString{String: hashedPassword, Valid: true}
	} else {
		hashedPasswordString = sql.NullString{Valid: false}
	}

	username := user.Username
	if username == "" {
		if strings.Contains(user.Email, "@") {
			username = user.Email[:strings.Index(user.Email, "@")]
		} else {
			username = user.Email
		}
	}

	_, err := r.store.CreateUser(ctx, db.CreateUserParams{
		ID:             user.ID,
		Name:           user.FullName,
		Username:       username,
		Email:          user.Email,
		Password:       hashedPasswordString,
		ProfilePicture: sql.NullString{String: user.ProfilePicture, Valid: user.ProfilePicture != ""},
		Bio:            sql.NullString{String: "", Valid: true},
		Role:           sql.NullString{String: user.Role, Valid: user.Role != ""},
		Phone:          sql.NullString{String: user.Phone, Valid: user.Phone != ""},
		Provider:       sql.NullString{String: provider, Valid: true},
		ProviderID:     providerIDNullable,
		EmailVerified:  sql.NullBool{Bool: false, Valid: true},
		IsActive:       sql.NullBool{Bool: true, Valid: true},
		LastLogin:      sql.NullTime{Valid: false},
	})

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password in the database.
func (r *UserRepository) UpdatePassword(ctx context.Context, email string, newPassword string) error {
	// Get the user first to verify they exist
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to retrieve user by email %s: %w", email, err)
	}

	// Hash the new password - move this to usecase layer for better separation of concerns
	// Here we assume the password is already hashed in the usecase layer

	// Update user's password
	_, err = r.store.UpdateUser(ctx, db.UpdateUserParams{
		ID: user.ID,
		Password: sql.NullString{
			String: newPassword,
			Valid:  true,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	userDetails, err := r.store.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	password := userDetails.Password.String
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty for user with ID %s", id)
	}

	return &entity.User{
		ID:        userDetails.ID,
		FullName:  userDetails.Name,
		Email:     userDetails.Email,
		CreatedAt: userDetails.CreatedAt.Time,
		Password:  password,
		Role:      userDetails.Role.String,
		Phone:     userDetails.Phone.String,
		UpdatedAt: userDetails.UpdatedAt.Time,
	}, nil

}

func (r *UserRepository) GetUserSession(ctx context.Context, sessionID uuid.UUID) (*entity.Session, error) {

	// Retrieve session details from the store
	sessionDetails, err := r.store.GetSessionByUserID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// // Convert sql.NullTime to *time.Time
	// var revokedAt *time.Time
	// if sessionDetails.RevokedAt.Valid {
	// 	revokedAt = &sessionDetails.RevokedAt.Time

	// }

	// Map session details to the entity.Session struct
	return &entity.Session{
		SessionID:    sessionDetails.SessionID,
		UserID:       sessionDetails.UserID,
		Token:        sessionDetails.Token,
		LastActivity: sessionDetails.LastActivity,
		ExpiresAt:    sessionDetails.ExpiresAt,
		CreatedAt:    sessionDetails.CreatedAt,
		DeviceInfo:   &sessionDetails.DeviceInfo,
		UserAgent:    sessionDetails.UserAgent.String,
		IpAddress:    sessionDetails.IpAddress.String,
		IsActive:     sessionDetails.IsActive,
		// RevokedAt:    revokedAt,
	}, nil
}

func (r *UserRepository) CreateSession(ctx context.Context, session *entity.Session) error {

	// Convert RevokedAt to sql.NullTime
	var revokedAt sql.NullTime
	if session.RevokedAt != nil {
		revokedAt = sql.NullTime{Time: *session.RevokedAt, Valid: true}
	} else {
		revokedAt = sql.NullTime{Valid: false}
	}

	// Call CreateSession with the mapped parameters
	_, err := r.store.CreateSession(ctx, db.CreateSessionParams{
		SessionID:    session.SessionID,
		UserID:       session.UserID,
		Token:        session.Token,
		LastActivity: session.LastActivity,
		ExpiresAt:    session.ExpiresAt,
		IpAddress:    sql.NullString{String: session.IpAddress, Valid: session.IpAddress != ""},
		UserAgent:    sql.NullString{String: session.UserAgent, Valid: session.UserAgent != ""},
		IsActive:     session.IsActive,
		RevokedAt:    revokedAt,
		DeviceInfo:   pqtype.NullRawMessage{},
	})
	if err != nil {
		return err
	}

	_, err = r.store.CreateEmailVerification(
		ctx,
		db.CreateEmailVerificationParams{
			ID:           uuid.New(),
			UserID:       session.UserID,
			OtpVerified:  sql.NullBool{Bool: session.OTPVerified, Valid: true},
			OtpExpiresAt: sql.NullTime{Time: session.OtpExpiresAt, Valid: true},
			Otp:          sql.NullString{String: session.Otp, Valid: session.Otp != ""},
			OtpAttempts:  sql.NullInt32{Int32: int32(session.OtpAttempts), Valid: true},
			ExpiresAt:    session.OtpExpiresAt,
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) UpdateSession(ctx context.Context, session *entity.Session) error {
	// Convert sessionID from string to UUID
	UserUUID, err := uuid.Parse(session.UserID.String())
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Call UpdateSession with the mapped parameters
	_, err = r.store.UpdateSession(ctx, db.UpdateSessionParams{
		UserID:       UserUUID,
		LastActivity: session.LastActivity,
		ExpiresAt:    session.ExpiresAt,
		IpAddress:    sql.NullString{String: session.IpAddress, Valid: session.IpAddress != ""},
		UserAgent:    sql.NullString{String: session.UserAgent, Valid: session.UserAgent != ""},
		DeviceInfo:   pqtype.NullRawMessage{RawMessage: session.DeviceInfo.RawMessage, Valid: session.DeviceInfo != nil},
		IsActive:     session.IsActive,
		RevokedAt:    sql.NullTime{Time: *session.RevokedAt, Valid: true},
	},
	)

	return err
}

// DeleteSession deletes a user session by its ID.
func (r *UserRepository) DeleteSession(ctx context.Context, id string) error {
	// Convert sessionID from string to UUID
	sessionUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Call DeleteSession with the mapped parameters
	err = r.store.DeleteSession(ctx, sessionUUID)

	return err
}

func (r *UserRepository) UpdateOtp(ctx context.Context, userOtp *entity.UpdateOtp) error {
	// Get User ID from the session
	user, err := r.GetUserByEmail(ctx, userOtp.Email)
	if err != nil {
		return err
	}

	if userOtp.OTPVerified {
		err = r.store.MarkEmailVerified(ctx, user.ID)
	} else {
		err = r.store.IncrementOTPAttempts(ctx, user.ID)
	}

	if err != nil {
		return err
	}

	return nil

}

func (r *UserRepository) GetOtp(ctx context.Context, email string) (*entity.UpdateOtp, error) {
	// Retrieve user by email
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user by email %s: %w", email, err)
	}

	// Convert userID from string to UUID
	userUUID, err := uuid.Parse(string(user.ID.String()))
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Call GetOtp with the mapped parameters
	otp, err := r.store.GetEmailVerification(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	// Map session details to the entity.Session struct
	return &entity.UpdateOtp{
		Otp:          otp.Otp.String,
		OtpExpiresAt: otp.OtpExpiresAt.Time,
		Email:        email,
		OTPVerified:  otp.OtpVerified.Bool,
		OtpAttempts:  int(otp.OtpAttempts.Int32),
	}, nil

}

// CreatePasswordReset stores a password reset token
func (r *UserRepository) CreatePasswordReset(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	// First, invalidate any existing reset tokens for this user
	err := r.DeletePasswordResetsByUserId(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate existing tokens: %w", err)
	}

	// Create new password reset token
	_, err = r.store.CreatePasswordReset(ctx, db.CreatePasswordResetParams{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	return nil
}

// GetPasswordResetByToken retrieves a password reset token and validates it
func (r *UserRepository) GetPasswordResetByToken(ctx context.Context, token string) (uuid.UUID, error) {
	// Get the token from the database - only valid and non-expired tokens
	resetToken, err := r.store.GetPasswordResetByToken(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid reset token: %w", err)
	}

	return resetToken.UserID, nil
}

// InvalidatePasswordReset marks a reset token as used
func (r *UserRepository) InvalidatePasswordReset(ctx context.Context, token string) error {
	_, err := r.store.InvalidatePasswordReset(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to invalidate reset token: %w", err)
	}

	return nil
}

// DeletePasswordResetsByUserId removes all password reset tokens for a user
func (r *UserRepository) DeletePasswordResetsByUserId(ctx context.Context, userID uuid.UUID) error {
	err := r.store.DeleteExpiredPasswordResets(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user's reset tokens: %w", err)
	}

	return nil
}

// GetUserSessions retrieves all sessions for a user
func (r *UserRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error) {
	// Check the return type of GetSessionsByUserID in your SQLC-generated code

	// If it returns []db.Session (a slice):
	dbSessions, err := r.store.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sessions: %w", err)
	}

	// Convert database sessions to entity sessions
	var result []*entity.Session
	for _, dbSession := range dbSessions {
		session := &entity.Session{
			SessionID:    dbSession.SessionID,
			UserID:       dbSession.UserID,
			Token:        dbSession.Token,
			CreatedAt:    dbSession.CreatedAt,
			ExpiresAt:    dbSession.ExpiresAt,
			LastActivity: dbSession.LastActivity,
			IpAddress:    dbSession.IpAddress.String,
			UserAgent:    dbSession.UserAgent.String,
			DeviceInfo:   &dbSession.DeviceInfo,
			IsActive:     dbSession.IsActive,
		}

		// Add RevokedAt if it exists
		if dbSession.RevokedAt.Valid {
			revokedAt := dbSession.RevokedAt.Time
			session.RevokedAt = &revokedAt
		}

		result = append(result, session)
	}

	return result, nil
}

// GetSessionByID retrieves a specific session by ID
func (r *UserRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*entity.Session, error) {
	// Get session from database
	session, err := r.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Map database session to entity session
	var revokedAt *time.Time
	if session.RevokedAt.Valid {
		revokedAt = &session.RevokedAt.Time
	}

	return &entity.Session{
		SessionID:    session.SessionID,
		UserID:       session.UserID,
		Token:        session.Token,
		CreatedAt:    session.CreatedAt,
		ExpiresAt:    session.ExpiresAt,
		LastActivity: session.LastActivity,
		IpAddress:    session.IpAddress.String,
		UserAgent:    session.UserAgent.String,
		DeviceInfo:   &session.DeviceInfo,
		IsActive:     session.IsActive,
		RevokedAt:    revokedAt,
	}, nil
}

// RevokeAllSessions revokes all active sessions for a user
func (r *UserRepository) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {

	// Call store method to revoke all sessions
	err := r.store.RevokeSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	return nil
}

// DeleteUser permanently deletes a user and all associated data
func (r *UserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// Delete user from database
	err := r.store.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetLoginHistory retrieves login history for a user
func (r *UserRepository) GetLoginHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.LoginHistoryEntry, error) {
	// Get login history from database
	history, err := r.store.GetLoginHistory(ctx, db.GetLoginHistoryParams{
		UserID: userID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve login history: %w", err)
	}

	// Map database login history to entity login history
	var result []*entity.LoginHistoryEntry
	for _, entry := range history {
		result = append(result, &entity.LoginHistoryEntry{
			ID:        entry.UserID,
			UserID:    entry.UserID,
			IpAddress: entry.IpAddress,
			UserAgent: entry.UserAgent,
		})
	}

	return result, nil
}
