package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/snagglebrew/shorten-url/database"
)

type UserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type UserRes struct {
	Username string `json:"username"`
	Secret   string `json:"secret"`
}

// Login is a function to login the user
func Login(c *fiber.Ctx) error {
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
	if r.SIsMember(database.Ctx, "users:public", req.Username).Val() {
		if r.HGet(database.Ctx, req.Username, "Password").Val() == req.Password {
			resp.Secret = r.HGet(database.Ctx, req.Username, "Secret").Val()
			if resp.Secret == "" {
				resp.Secret = uuid.New().String()
				r.HSet(database.Ctx, req.Username, "Secret", resp.Secret)
			}
			return c.Status(fiber.StatusOK).JSON(resp)
		}
	}
	return c.Status(fiber.StatusUnauthorized).JSON(resp)
}
