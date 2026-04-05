-- name: CreateEmailVerification :one
INSERT INTO email_verification (
    id, 
    user_id, 
    otp, 
    otp_verified,
    otp_attempts,
    otp_expires_at, 
    created_at, 
    expires_at, 
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now(), $7, now()
) RETURNING *;

-- name: GetEmailVerification :one
SELECT * FROM email_verification
WHERE user_id = $1
LIMIT 1;

-- name: MarkEmailVerified :exec
UPDATE email_verification
SET otp_verified = true,
    updated_at = now()
WHERE user_id = $1;

-- name: IncrementOTPAttempts :exec
UPDATE email_verification
SET otp_attempts = otp_attempts + 1,
    updated_at = now()
WHERE user_id = $1;

-- name: DeleteExpiredEmailVerifications :exec
DELETE FROM email_verification
WHERE expires_at < now();
