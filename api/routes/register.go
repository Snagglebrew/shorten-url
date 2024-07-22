package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/snagglebrew/shorten-url/database"
)

// Register is a function to login the user
// UserReq is a struct to hold the request body, stored in login.go
// Use Query ?public=true in POST to make the user allowed to post public URLs
func Register(c *fiber.Ctx) error {
	req := new(UserReq)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	r := database.CreateClient(0)
	defer r.Close()

	resp := UserRes{
		Username: req.Username,
		Secret:   "",
	}
	if !r.SIsMember(database.Ctx, "users:public", req.Username).Val() {
		if req.Password != "" {
			resp.Secret = uuid.New().String()
			r.HSet(database.Ctx, req.Username, "Password", req.Password, "Secret", resp.Secret)
			if c.Query("public") == "true" {
				r.SAdd(database.Ctx, "users:public", req.Username)
			}
		}
		return c.Status(fiber.StatusOK).JSON(resp)
	}
	return c.Status(fiber.StatusUnauthorized).JSON(resp)
}
