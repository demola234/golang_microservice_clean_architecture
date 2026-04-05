package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/demola234/authentication/internal/domain/entity"
	"github.com/demola234/authentication/internal/domain/repository"
	"github.com/demola234/authentication/pkg/utils"
	"github.com/demola234/authentication/pkg/val"

	"github.com/google/uuid"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserUsecase defines the interface for user-related business logic.
type UserUsecase interface {
	RegisterUser(ctx context.Context, fullName string, password string, email string, role string, phone string) (*entity.User, *entity.Session, error)
	LoginUser(ctx context.Context, password, email string) (*entity.User, error)
	ChangePassword(ctx context.Context, currentPassword, newPassword, id string) error
	GetSession(ctx context.Context, id string) (*entity.Session, error)
	GenerateToken(ctx context.Context, email string, userID string) (string, error)
	ResendOtp(ctx context.Context, email string) error
	GetUser(ctx context.Context, userId string) (*entity.User, error)
	LogOut(ctx context.Context, userId string) error
	VerifyOtp(ctx context.Context, email string, otp string) (bool, error)
	RegisterWithOAuth(ctx context.Context, provider, token string) (*entity.User, *entity.Session, error)
	LoginWithOAuth(ctx context.Context, provider, token string) (*entity.User, *entity.Session, error)
	UppdateProfileImage(ctx context.Context, content io.Reader, userId uuid.UUID) (string, error)
	ForgetPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
	VerifyResetPassword(ctx context.Context, email string, otp string) error
	GetUserProfile(ctx context.Context, userID string) (*entity.UserProfile, error)
	UpdateUserProfile(ctx context.Context, profile *entity.UserProfile, userID string) (*entity.UserProfile, error)
	GetSessions(ctx context.Context, userID string) ([]*entity.Session, error)
	RevokeSession(ctx context.Context, sessionID string, userID string) error
	DeactivateAccount(ctx context.Context, password string, userID string) error
	DeleteAccount(ctx context.Context, password string, userID string) error
	GetLoginHistory(ctx context.Context, userID string, limit int) ([]*entity.LoginHistoryEntry, error)
}

// userUsecase implements the UserUsecase interface.
type userUsecase struct {
	userRepo  repository.UserRepository
	oauthRepo repository.OAuthRepository
}

// RegisterWithOAuth implements UserUsecase.
func (u *userUsecase) RegisterWithOAuth(ctx context.Context, provider string, token string) (*entity.User, *entity.Session, error) {
	userInfo, err := u.oauthRepo.ValidateProviderToken(ctx, provider, token)
	userID := uuid.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to validate OAuth token: %w", err)
	}

	// Check if the user already exists
	existingUser, err := u.userRepo.GetUserByEmail(ctx, userInfo.Email)
	if err == nil && existingUser != nil {
		currentProvider, _ := existingUser.Provider.GetProviderInfo()
		if currentProvider != provider {
			// Update to new provider
			existingUser.Provider.SetProvider(provider, userInfo.ID)
			existingUser.ProviderID = userInfo.ID

		}
	}

	metaData := utils.ExtractMetaData(ctx)

	session := &entity.Session{
		SessionID:    uuid.New(),
		UserID:       userID,
		Token:        token,
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().Add(24 * time.Hour).UTC(),
		LastActivity: time.Now().UTC(),
		IpAddress:    metaData.ClientIP,
		UserAgent:    metaData.UserAgent,
		IsActive:     true,
		OTPVerified:  true,
		OtpExpiresAt: time.Now().Add(5 * time.Minute),
	}

	userDetails := &entity.User{
		FullName: userInfo.Name,
		Email:    userInfo.Email,
		Phone:    "",
		Role:     "user",
		ID:       userID,
		Bio:      "",
		Provider: utils.ProviderType{
			Name:      provider,
			ID:        userInfo.ID,
			TokenData: provider,
		},
		ProviderID:     userInfo.ID,
		Username:       userInfo.Name,
		ProfilePicture: userInfo.Picture,
		EmailVerified:  userInfo.EmailVerified,
		IsActive:       true,
		LastLogin:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Save user in the repository
	if err := u.userRepo.CreateUser(ctx, userDetails); err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save session in the repository
	if err := u.userRepo.CreateSession(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return existingUser, session, nil
}

// NewUserUsecase creates a new instance of userUsecase.
func NewUserUsecase(userRepo repository.UserRepository, oauthRepo repository.OAuthRepository) UserUsecase {
	return &userUsecase{userRepo: userRepo, oauthRepo: oauthRepo}
}

func (u *userUsecase) UppdateProfileImage(ctx context.Context, content io.Reader, userId uuid.UUID) (string, error) {

	uploadedImageUrl, err := u.userRepo.UploadProfileImage(ctx, content, userId)
	if err != nil {
		return "", fmt.Errorf("failed to upload profile image: %w", err)
	}

	return uploadedImageUrl, nil
}

// GenerateToken implements UserUsecase.
func (u *userUsecase) GenerateToken(ctx context.Context, email string, userID string) (string, error) {
	token, err := u.userRepo.CreateToken(ctx, email, userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token for email %s: %w", email, err)
	}
	return token, nil
}

// RegisterUser registers a new user.
func (u *userUsecase) RegisterUser(ctx context.Context, fullName, password, email, role, phone string) (*entity.User, *entity.Session, error) {

	// Check if user with the same email already exists
	existingUser, err := u.userRepo.GetUserByEmail(ctx, email)
	userID := uuid.New()
	if err == nil && existingUser != nil {
		return nil, nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Generate a new token for the session
	token, err := u.userRepo.CreateToken(ctx, email, userID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate token for email %s: %w", email, err)
	}

	// Create a new user entity with a unique ID
	user := &entity.User{
		ID:       userID,
		Email:    email,
		FullName: fullName,
		Password: password,
		Role:     role,
		Phone:    phone,
	}

	// Gather metadata for session
	metaData := utils.ExtractMetaData(ctx)

	// Create a new session entity with a unique ID and expiration time
	session := &entity.Session{
		SessionID:    uuid.New(),
		UserID:       user.ID,
		Token:        token,
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().Add(24 * time.Hour).UTC(),
		LastActivity: time.Now().UTC(),
		IpAddress:    metaData.ClientIP,
		UserAgent:    metaData.UserAgent,
		IsActive:     true,
		Otp:          utils.RandomOtp(),
		OTPVerified:  false,
		OtpExpiresAt: time.Now().Add(5 * time.Minute),
		OtpAttempts:  0,
	}

	// Save user in the repository
	if err := u.userRepo.CreateUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save session in the repository
	if err := u.userRepo.CreateSession(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return user, session, nil
}

// LoginUser authenticates a user by email and password.
func (u *userUsecase) LoginUser(ctx context.Context, password string, email string) (*entity.User, error) {
	// Retrieve user by email
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Check if the provided password matches the stored hash
	err = utils.CheckPassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword updates a user's password.
func (u *userUsecase) ChangePassword(ctx context.Context, currentPassword string, newPassword string, email string) error {
	// Retrieve the user by ID to verify the current password
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to retrieve user by ID %s: %w", email, err)
	}

	// Check if current password is correct
	if err := utils.CheckPassword(currentPassword, user.Password); err != nil {
		return fmt.Errorf("current password is incorrect: %w", err)
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	session, err := u.userRepo.GetUserSession(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user session: %w", err)
	}

	// Check if the user is active
	if !session.IsActive {
		return fmt.Errorf("user is not active")
	}

	// Check if user is verified
	if !session.OTPVerified {
		return fmt.Errorf("user is not verified")
	}

	// Update the password in the repository
	err = u.userRepo.UpdatePassword(ctx, email, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUser implements UserUsecase.
func (u *userUsecase) GetUser(ctx context.Context, userId string) (*entity.User, error) {
	// Retrieve user by email
	user, err := u.userRepo.GetUserByID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user by userId %s: %w", userId, err)
	}

	return user, nil
}

// LogOut implements UserUsecase.
func (u *userUsecase) LogOut(ctx context.Context, userId string) error {
	// Retrieve user by email
	err := u.userRepo.DeleteSession(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to retrieve user by userId %s: %w", userId, err)
	}

	return nil
}

// ResendOtp implements UserUsecase.
func (u *userUsecase) ResendOtp(ctx context.Context, email string) error {
	// Retrieve user by email
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to retrieve user by email %s: %w", email, err)
	}

	//

	// Retrieve user by email
	err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
		Otp:          utils.RandomOtp(),
		OtpAttempts:  0,
		Email:        user.Email,
		OtpExpiresAt: time.Now().Add(time.Minute * 10),
	})

	if err != nil {
		return fmt.Errorf("failed to update otp: %w", err)
	}

	return nil
}

// GetSession implements UserUsecase.
func (u *userUsecase) GetSession(ctx context.Context, id string) (*entity.Session, error) {
	userId, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}
	// Retrieve user by email
	session, err := u.userRepo.GetUserSession(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user by userId %s: %w", id, err)
	}

	return session, nil

}

// VerifyOtp implements UserUsecase.
func (u *userUsecase) VerifyOtp(ctx context.Context, email string, otp string) (bool, error) {
	// Retrieve user by email
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve user by email %s: %w", email, err)
	}

	otpUpdate, err := u.userRepo.GetOtp(ctx, user.ID.String())
	if err != nil {
		return false, fmt.Errorf("failed to retrieve user session: %w", err)
	}

	if otpUpdate.OTPVerified {
		return false, fmt.Errorf("otp already verified")
	}

	if otpUpdate.OtpAttempts >= 10 {
		return false, fmt.Errorf("otp attempts exceeded")
	}

	if otpUpdate.Otp != otp {
		return false, fmt.Errorf("invalid otp %s: %s", otp, otpUpdate.Otp)
	}

	session, err := u.userRepo.GetUserSession(ctx, user.ID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve user session: %w", err)
	}

	if session.OtpExpiresAt.After(time.Now()) {
		return false, fmt.Errorf("otp expired")
	}

	if session.OtpAttempts >= 10 {
		return false, fmt.Errorf("otp attempts exceeded")
	}

	// Update the password in the repository
	err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
		OtpAttempts: session.OtpAttempts + 1,
		Email:       user.Email,
		OTPVerified: true,
	})

	if err != nil {
		return false, fmt.Errorf("failed to update password: %w", err)
	}

	return true, nil

}

// ForgetPassword implements UserUsecase with 6-digit OTP
func (u *userUsecase) ForgetPassword(ctx context.Context, email string) error {
	// Check if the user exists
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil
	}

	// Generate a secure 6-digit OTP
	otp := utils.RandomOtp()

	// Update the OTP for the user - reusing your existing UpdateOtp method
	err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
		Otp:          otp,
		OtpAttempts:  0,
		Email:        user.Email,
		OTPVerified:  false,
		OtpExpiresAt: time.Now().Add(10 * time.Minute), // OTP valid for 10 minutes
	})

	if err != nil {
		return fmt.Errorf("failed to update OTP: %w", err)
	}

	// Get metadata for logging
	metaData := utils.ExtractMetaData(ctx)

	// For development, log the OTP
	fmt.Printf("[Password Reset] User: %s, IP: %s, OTP: %s\n",
		user.Email, metaData.ClientIP, otp)
	return nil
}

// VerifyResetPassword verifies the OTP for password rese
// VerifyResetPassword verifies the OTP for password reset
func (u *userUsecase) VerifyResetPassword(ctx context.Context, email string, otp string) error {
	// Validate OTP format
	if !utils.ValidateOTP(otp) {
		return fmt.Errorf("invalid OTP format")
	}

	// Check if the user exists
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Get the current OTP details using your existing GetOtp method
	otpDetails, err := u.userRepo.GetOtp(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to retrieve OTP details: %w", err)
	}

	// Check if OTP is already verified
	if otpDetails.OTPVerified {
		return fmt.Errorf("OTP already verified")
	}

	// Check if max attempts exceeded
	if otpDetails.OtpAttempts >= 10 {
		return fmt.Errorf("maximum OTP attempts exceeded")
	}

	// Check if OTP has expired
	if time.Now().After(otpDetails.OtpExpiresAt) {
		return fmt.Errorf("OTP has expired")
	}

	// Check if OTP matches
	if otpDetails.Otp != otp {
		// Increment attempts and update
		err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
			Email:       user.Email,
			OtpAttempts: otpDetails.OtpAttempts + 1,
		})
		if err != nil {
			return fmt.Errorf("failed to update OTP attempts: %w", err)
		}
		return fmt.Errorf("invalid OTP")
	}

	// OTP is valid, mark as verified
	err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
		Email:       user.Email,
		OTPVerified: true,
	})
	if err != nil {
		return fmt.Errorf("failed to mark OTP as verified: %w", err)
	}

	return nil
}

// ResetPassword sets a new password after OTP verification
func (u *userUsecase) ResetPassword(ctx context.Context, email string, newPassword string) error {
	// Check if the user exists
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Get the session to check if OTP was verified
	session, err := u.userRepo.GetUserSession(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user session: %w", err)
	}

	// Check if OTP was verified
	if !session.OTPVerified {
		return fmt.Errorf("OTP not verified, cannot reset password")
	}

	// Validate the new password
	if err := val.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update the password
	err = u.userRepo.UpdatePassword(ctx, email, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Reset the OTP verification status
	err = u.userRepo.UpdateOtp(ctx, &entity.UpdateOtp{
		Email:       user.Email,
		OTPVerified: false,
	})
	if err != nil {
		return fmt.Errorf("failed to reset OTP verification status: %w", err)
	}

	// Get metadata for logging
	metaData := utils.ExtractMetaData(ctx)

	// Log the password change
	fmt.Printf("[Password Changed] User: %s, IP: %s\n",
		user.Email, metaData.ClientIP)

	return nil
}

// UpdateUserProfile updates a user's profile information
func (u *userUsecase) UpdateUserProfile(ctx context.Context, profile *entity.UserProfile, userID string) (*entity.UserProfile, error) {
	// Retrieve the user
	user, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Update user fields
	user.FullName = profile.User.FullName
	user.Bio = profile.Bio
	user.Phone = profile.User.Phone
	// Add any other fields that should be updated

	// Update user in repository
	err = u.userRepo.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return the updated profile
	return &entity.UserProfile{
		User:     user,
		Bio:      user.Bio,
		Website:  profile.Website,
		Location: profile.Location,
		JoinedAt: user.CreatedAt,
	}, nil
}

// GetSessions retrieves all active sessions for a user
func (u *userUsecase) GetSessions(ctx context.Context, userID string) ([]*entity.Session, error) {
	// Parse the user ID
	userId, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Retrieve sessions from repository
	// Note: You would need to add a method to your repository to get all sessions
	sessions, err := u.userRepo.GetUserSessions(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sessions: %w", err)
	}

	return sessions, nil
}

// RevokeSession revokes a specific session
func (u *userUsecase) RevokeSession(ctx context.Context, sessionID string, userID string) error {
	// Parse the session ID
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Check if the session belongs to the user
	session, err := u.userRepo.GetSessionByID(ctx, sessID)
	if err != nil {
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Verify that the session belongs to the requesting user
	userId, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	if session.UserID != userId {
		return fmt.Errorf("unauthorized: session does not belong to the user")
	}

	// Revoke the session
	// This might involve setting it as inactive, setting a revoked time, etc.
	// We'll add a RevokedAt field and set IsActive to false
	revokedAt := time.Now().UTC()
	session.RevokedAt = &revokedAt
	session.IsActive = false

	// Update the session in the repository
	err = u.userRepo.UpdateSession(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// DeactivateAccount temporarily deactivates a user account
func (u *userUsecase) DeactivateAccount(ctx context.Context, password string, userID string) error {
	// Retrieve the user
	user, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify password
	err = utils.CheckPassword(password, user.Password)
	if err != nil {
		return fmt.Errorf("password is incorrect: %w", err)
	}

	// Deactivate the account (set IsActive to false)
	user.IsActive = false

	// Update user in repository
	err = u.userRepo.UpdateUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to deactivate account: %w", err)
	}

	// Revoke all active sessions
	userId, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	err = u.userRepo.RevokeAllSessions(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	// Log the account deactivation
	metaData := utils.ExtractMetaData(ctx)
	fmt.Printf("Account deactivated for user %s from IP %s at %s\n",
		user.Email, metaData.ClientIP, time.Now().Format(time.RFC3339))

	return nil
}

// DeleteAccount permanently deletes a user account
func (u *userUsecase) DeleteAccount(ctx context.Context, password string, userID string) error {
	// Retrieve the user
	user, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify password
	err = utils.CheckPassword(password, user.Password)
	if err != nil {
		return fmt.Errorf("password is incorrect: %w", err)
	}

	// Parse the user ID
	userId, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Delete user from repository
	err = u.userRepo.DeleteUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Log the account deletion
	metaData := utils.ExtractMetaData(ctx)
	fmt.Printf("Account deleted for user %s from IP %s at %s\n",
		user.Email, metaData.ClientIP, time.Now().Format(time.RFC3339))

	return nil
}

// GetLoginHistory retrieves the login history for a user
func (u *userUsecase) GetLoginHistory(ctx context.Context, userID string, limit int) ([]*entity.LoginHistoryEntry, error) {
	// Parse the user ID
	userId, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Retrieve login history from repository
	// Note: You would need to add a method to your repository to get login history
	history, err := u.userRepo.GetLoginHistory(ctx, userId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve login history: %w", err)
	}

	return history, nil
}

// GetUserProfile retrieves a user's profile
func (u *userUsecase) GetUserProfile(ctx context.Context, userID string) (*entity.UserProfile, error) {
	// Retrieve the user
	user, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	profile := &entity.UserProfile{
		User:     user,
		Bio:      user.Bio,
		Website:  "",
		Location: "",
		JoinedAt: user.CreatedAt,
	}

	return profile, nil
}

// LoginWithOAuth implements UserUsecase.
func (u *userUsecase) LoginWithOAuth(ctx context.Context, provider string, token string) (*entity.User, *entity.Session, error) {
	panic("unimplemented")
}
