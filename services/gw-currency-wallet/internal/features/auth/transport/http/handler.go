package auth_http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"wallet-app/internal/core/domain"
)

type Handler struct {
	authService AuthService
}

func New(authService AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

// POST /api/v1/register
//
// request body:
//
//	{
//		"username": "client_nickname",
//		"email": "some@gmail.com",
//		"password": "12345678"
//	}
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	var registerDTO RegisterDTO

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&registerDTO)
	if err == nil {
		var extra any
		if nextErr := decoder.Decode(&extra); nextErr != io.EOF {
			if nextErr == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = nextErr
			}
		}
	}

	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		status := http.StatusBadRequest
		message := ErrorInvalidRequestBody

		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			status = http.StatusRequestEntityTooLarge
			message = ErrorRequestBodyTooLarge
		}

		writeJSON(w, status, ErrorResponse{
			Error: message,
		})
		return
	}

	user, err := h.authService.Register(r.Context(), registerDTO.Username, registerDTO.Email, registerDTO.Password)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		var response any
		var statusCode int

		if errors.Is(err, domain.ErrInvalidUsername) {
			response = ErrorResponse{
				Error: ErrorInvalidUsername,
			}
			statusCode = http.StatusBadRequest
		} else if errors.Is(err, domain.ErrInvalidEmailAddress) {
			response = ErrorResponse{
				Error: ErrorInvalidEmail,
			}
			statusCode = http.StatusBadRequest
		} else if errors.Is(err, domain.ErrInvalidPassword) {
			response = ErrorResponse{
				Error: ErrorInvalidPassword,
			}
			statusCode = http.StatusBadRequest
		} else if errors.Is(err, domain.ErrUsernameAlreadyExists) || errors.Is(err, domain.ErrEmailAlreadyExists) {
			response = ErrorResponse{
				Error: ErrorUsernameOrEmailAlreadyExists,
			}
			statusCode = http.StatusBadRequest
		} else {
			response = ErrorResponse{
				Error: ErrorInternalServerError,
			}
			statusCode = http.StatusInternalServerError
		}

		level := slog.LevelWarn
		if statusCode >= 500 {
			level = slog.LevelError
		}
		slog.Log(r.Context(), level, "user registration failed", "error", err)
		writeJSON(w, statusCode, response)
		return
	}

	slog.InfoContext(r.Context(), "new user registred", "user_id", user.ID().String(), "email", user.Email(), "username", user.Username())
	writeJSON(w, http.StatusCreated, RegistrationSuccessResponse{
		Message: "User registered successfully",
	})
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	body, err := json.Marshal(response)
	if err != nil {
		slog.Error("marshal response failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		slog.Warn("write response failed", "error", err)
	}
}

// POST /api/v1/login
//
// request body:
//
//	{
//		"username": "client_nickname",
//		"password": "12345678"
//	}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	var loginDTO LoginDTO

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&loginDTO)
	if err == nil {
		var extra any
		if nextErr := decoder.Decode(&extra); nextErr != io.EOF {
			if nextErr == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = nextErr
			}
		}
	}

	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		status := http.StatusBadRequest
		message := ErrorInvalidRequestBody

		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			status = http.StatusRequestEntityTooLarge
			message = ErrorRequestBodyTooLarge
		}

		writeJSON(w, status, ErrorResponse{
			Error: message,
		})
		return
	}

	user, token, err := h.authService.Login(r.Context(), loginDTO.Username, loginDTO.Password)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		var response any
		var statusCode int

		if errors.Is(err, domain.ErrInvalidUserCredentials) || errors.Is(err, domain.ErrInvalidPassword) {
			response = ErrorResponse{
				Error: ErrorInvalidUserCredentials,
			}
			statusCode = http.StatusBadRequest
		} else {
			response = ErrorResponse{
				Error: ErrorInternalServerError,
			}
			statusCode = http.StatusInternalServerError
		}

		level := slog.LevelWarn
		if statusCode >= 500 {
			level = slog.LevelError
		}
		slog.Log(r.Context(), level, "user login failed", "error", err)
		writeJSON(w, statusCode, response)
		return
	}

	slog.InfoContext(r.Context(), "login", "user_id", user.ID().String(), "email", user.Email(), "username", user.Username())
	writeJSON(w, http.StatusCreated, LoginSuccessResponse{
		Token: token,
	})
}
