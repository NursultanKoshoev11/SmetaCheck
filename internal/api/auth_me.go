package api

import "net/http"

func AuthMe(w http.ResponseWriter, r *http.Request) {
	requestUser, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}
	var user User
	err = pool.QueryRow(r.Context(), `
		SELECT id,COALESCE(email,''),COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at
		FROM users WHERE id=$1
	`, requestUser.ID).Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.EmailVerifiedAt,&user.CreatedAt)
	if err != nil {
		estimateWriteError(w, http.StatusUnauthorized, "user session is invalid")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"user": user})
}
