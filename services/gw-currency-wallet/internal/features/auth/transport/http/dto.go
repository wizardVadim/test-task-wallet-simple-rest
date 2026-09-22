package auth_http

type RegisterDTO struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegistrationSuccessResponse struct {
	Message string `json:"message"`
}

type ErrorMessage string

const (
	ErrorInvalidRequestBody           ErrorMessage = "invalid request body"
	ErrorRequestBodyTooLarge          ErrorMessage = "request body too large"
	ErrorInvalidUsername              ErrorMessage = "invalid username"
	ErrorInvalidEmail                 ErrorMessage = "invalid user email"
	ErrorInvalidPassword              ErrorMessage = "invalid user password"
	ErrorInternalServerError          ErrorMessage = "internal server error"
	ErrorUsernameOrEmailAlreadyExists ErrorMessage = "Username or email already exists"
	ErrorInvalidUserCredentials       ErrorMessage = "Invalid username or password"
)

type LoginDTO struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginSuccessResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error ErrorMessage `json:"error"`
}
