package main

import (
	"net/http"
	"time"
	"log"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"strings"
	"os"
	"database/sql"
	"github.com/joho/godotenv"
	"github.com/JackFretwell/chirpy/internal/database"
	"github.com/JackFretwell/chirpy/internal/auth"
	"github.com/google/uuid"
)

import _ "github.com/lib/pq"

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secret string
}

type User struct {
	ID				uuid.UUID `json:"id"`
	CreatedAt		time.Time `json:"created_at"`
	UpdatedAt		time.Time `json:"updated_at"`
	Email			string	  `json:"email"`
	Token			string	  `json:"token"`
	RefreshToken	string	  `json:"refresh_token"`
}

type Chirp struct {
	ID			uuid.UUID `json:"id"`
	CreatedAt	time.Time `json:"created_at"`
	UpdatedAt	time.Time `json:"updated_at"`
	Body		string	  `json:"body"`
	UserID		uuid.UUID `json:"user_id"`
}

type createUserParams struct {
	Password 			string 		`json:"password"`
	Email				string		`json:"email"`
}


func healthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) writeNumberOfRequests(w http.ResponseWriter, req *http.Request) {
	hits := cfg.fileserverHits.Load()
	formattedText := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`,
	hits)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(formattedText))
}

func (cfg *apiConfig) resetFileserverHits (w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	if cfg.platform == "dev"{
		err := cfg.dbQueries.DeleteUsers(req.Context())
		if err != nil {
			respondWithError(w, 400, "An occured when deleting users from the database")
			return
		}
	} else {
		respondWithError(w, 403, "This command is forbidden in a production environment")
		return
	}

}

func profanityFilter(s string) string {
	splitString := strings.Split(s, " ")
	for i := 0; i < len(splitString); i++ {
		if strings.ToLower(splitString[i]) == "kerfuffle" || strings.ToLower(splitString[i]) == "sharbert" || strings.ToLower(splitString[i]) == "fornax" {
			splitString[i] = "****"
		}
	}

	return strings.Join(splitString, " ")
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	type chirpValid struct {
		Body 	string 		`json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	c := chirpValid{}
	err := decoder.Decode(&c)
	if err != nil {
		respondWithError(w, 400, "An error occured when decoding the Chirp")
		return
	}

	if len(c.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	headers := req.Header
	token, err := auth.GetBearerToken(headers)
	if err != nil {
		respondWithError(w, 400, "An error occured when retrieving the bearer token")
		return
	}

	id, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "An error occured when validating the bearer token")
		return
	}

	cleanText := profanityFilter(c.Body)

	params := database.CreateChirpParams{
		Body: cleanText,
		UserID: id,
	}

	createdChirp, err := cfg.dbQueries.CreateChirp(req.Context(), params)
	if err != nil {
		respondWithError(w, 400, "An occured when creating the Chirp in our database")
		return
	} 

	respBody := Chirp{
		ID: 	   createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body:	   createdChirp.Body,
		UserID:	   id,
	}

	respondWithJSON(w, 201, respBody)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
		Error string `json:"error"`
	}

	respBody := returnVals{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	c := createUserParams{}
	err := decoder.Decode(&c)
	if err != nil {
		respondWithError(w, 400, "An error occured when decoding the user's email")
		return
	}

	hash, err := auth.HashPassword(c.Password)
	if err != nil {
		respondWithError(w, 400, "An error occured when hashing the user's password")
		return
	}

	params := database.CreateUserParams{
		Email: c.Email,
		HashedPassword: hash,
	}

	createdUser, err := cfg.dbQueries.CreateUser(req.Context(), params)
	if err != nil {
		respondWithError(w, 400, "An error occured when creating the user")
		return
	}

	u := User{
		ID:		   		createdUser.ID,
		CreatedAt: 		createdUser.CreatedAt,
		UpdatedAt: 		createdUser.UpdatedAt,
		Email:	   		createdUser.Email,
	}

	respondWithJSON(w, 201, u)
}

func (cfg *apiConfig) retrieveChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.RetrieveChirps(req.Context())
	if err != nil {
		respondWithError(w, 400, "An error occured when retrieving chirps")
		return
	}

	chirpArray := make([]Chirp, len(chirps))

	for i := 0; i < len(chirps); i++ {
		c := Chirp{
			ID: 	   chirps[i].ID,
			CreatedAt: chirps[i].CreatedAt,
			UpdatedAt: chirps[i].UpdatedAt,
			Body:	   chirps[i].Body,
			UserID:	   chirps[i].UserID,
		}
		chirpArray[i] = c
	}
	
	respondWithJSON(w, 200, chirpArray)
}

func (cfg *apiConfig) retrieveSpecificChirp(w http.ResponseWriter, req *http.Request){
	chirpID := req.PathValue("chirpID")
	uuid, _ := uuid.Parse(chirpID)
	chirp, err := cfg.dbQueries.RetrieveChirp(req.Context(), uuid)
	if err != nil {
		respondWithError(w, 404, "An error occured when retrieving the given chirp")
		return
	}

	c := Chirp{
		ID: 	   chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:	   chirp.Body,
		UserID:	   chirp.UserID,
	}

	respondWithJSON(w, 200, c)
}

func (cfg *apiConfig) userLogin(w http.ResponseWriter, req *http.Request){
	decoder := json.NewDecoder(req.Body)
	c := createUserParams{}
	err := decoder.Decode(&c)
	if err != nil {
		respondWithError(w, 400, "An error occured when decoding the user's email")
		return
	}
	
	user, err := cfg.dbQueries.FindUserByEmail(req.Context(), c.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(c.Password, user.HashedPassword)
	if !match {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	jwt, err := auth.MakeJWT(user.ID, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Error occurred creating JWT")
		return
	}

	refreshToken := auth.MakeRefreshToken()
	params := database.CreateRefreshTokenParams{
		Token: refreshToken,
		UserID: user.ID,
	}

	_, err = cfg.dbQueries.CreateRefreshToken(req.Context(), params)
	if err != nil {
		respondWithError(w, 401, "Error occurred creating refresh token")
		return
	}

	u := User{
		ID:		   		user.ID,
		CreatedAt: 		user.CreatedAt,
		UpdatedAt: 		user.UpdatedAt,
		Email:	   		user.Email,
		Token:			jwt,
		RefreshToken:   refreshToken,
	}
	respondWithJSON(w, 200, u)
}

func (cfg *apiConfig) refresh(w http.ResponseWriter, req *http.Request) {
	type refreshToken struct{
		Token	string	`json:"token"`
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "An error occured when retrieving the bearer token")
		return
	}


	rToken, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(w, 401, "Could not find the given refresh token in our database")
		return
	}

	if rToken.ExpiresAt.Compare(time.Now()) < 0 || rToken.RevokedAt.Valid {
		respondWithError(w, 401, "The given token has expired or been revoked")
		return
	}

	jwt, err := auth.MakeJWT(rToken.UserID, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Error occurred creating JWT")
		return
	}

	r := refreshToken{
		Token: jwt,
	}

	respondWithJSON(w, 200, r)
 
}

func (cfg *apiConfig) revoke(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "An error occured when retrieving the bearer token")
		return
	}

	refreshToken, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(w, 401, "Could not find the given refresh token in our database")
		return
	}

	err = cfg.dbQueries.UpdateRevokedAtField(req.Context(), refreshToken.Token)
	if err != nil {
		respondWithError(w, 401, "Could not successfully revoke the refresh token")
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) userUpdate(w http.ResponseWriter, req *http.Request) {
	type updateRequest struct {
		Password	string		`json:"password"`
		Email		string		`json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	updateReq := updateRequest{}
	err := decoder.Decode(&updateReq)
	if err != nil {
		respondWithError(w, 400, "An error occured when decoding the user update request body")
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "An error occured when retrieving the bearer token")
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unable to validate the users access token")
		return
	}

	hash, err := auth.HashPassword(updateReq.Password)
	if err != nil {
		respondWithError(w, 400, "An error occured when hashing the given password")
		return
	}

	params := database.UpdateUsersEmailPasswordParams{
		Email: updateReq.Email,
		HashedPassword: hash,
		ID: userID,
	}

	user, err := cfg.dbQueries.UpdateUsersEmailPassword(req.Context(), params)
	if err != nil {
		respondWithError(w, 400, "An error occured when updating the database record")
		return
	}

	u := User{
		ID:		   		user.ID,
		CreatedAt: 		user.CreatedAt,
		UpdatedAt: 		user.UpdatedAt,
		Email:	   		user.Email,
	}

	respondWithJSON(w, 200, u)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "An error occured when retrieving the bearer token")
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unable to validate the users access token")
		return
	}

	chirpID := req.PathValue("chirpID")
	uuid, _ := uuid.Parse(chirpID)

	chirp, err := cfg.dbQueries.RetrieveChirp(req.Context(), uuid)
	if err != nil {
		respondWithError(w, 404, "Chirp does not exist")
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, 403, "This user is not the author of the given chirp")
		return
	}

	err = cfg.dbQueries.DeleteChirp(req.Context(), uuid)
	if err != nil {
		respondWithError(w, 400, "An error occurred when deleting the chirp")
		return
	}

	w.WriteHeader(204)
}


func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("An error occurred when opening the database", err)
	}
	dbQueries := database.New(db)

	cfg := apiConfig{}
	cfg.dbQueries = dbQueries
	cfg.platform = os.Getenv("PLATFORM")
	cfg.secret = os.Getenv("SECRET")
	
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	mux.HandleFunc("POST /api/chirps", cfg.createChirp)
	mux.HandleFunc("GET /api/chirps", cfg.retrieveChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.retrieveSpecificChirp)
	mux.HandleFunc("POST /api/login", cfg.userLogin)
	mux.HandleFunc("POST /api/refresh", cfg.refresh)
	mux.HandleFunc("POST /api/revoke", cfg.revoke)
	mux.HandleFunc("PUT /api/users", cfg.userUpdate)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirp)

	mux.HandleFunc("POST /admin/reset", cfg.resetFileserverHits)
	mux.HandleFunc("GET /admin/metrics", cfg.writeNumberOfRequests)

	s := &http.Server {
		Addr:			":8080",
		Handler:		mux,
		ReadTimeout:	10 * time.Second,
		WriteTimeout:	10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	//log.Printf("Serving files from %s on port: %s\n", )
	log.Fatal(s.ListenAndServe())
}
