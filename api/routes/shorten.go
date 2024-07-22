package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/snagglebrew/shorten-url/database"
	"github.com/snagglebrew/shorten-url/helpers"
)

type URLReq struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
	Public      bool          `json:"public"`
	Username    string        `json:"username"`
	SecretKey   string        `json:"secret"`
}

type UrlRes struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"X-Rate-Limit-Remaining"`
	XRateLimitReset time.Duration `json:"X-Rate-Limit-Reset"`
}

/***
** @Description : ShortenURL is a function to shorten the req URL
** r := database.CreateClient(1) Rate limiting
** r2 := database.CreateClient(2) Shortened URL storage
***/
func ShortenURL(c *fiber.Ctx) error {
	req := new(URLReq)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	rateQuota, _ := strconv.Atoi(os.Getenv("API_QUOTA"))
	duration, _ := time.ParseDuration(os.Getenv("API_QUOTA_DURATION") + "h")

	//Rate Limiting
	r := database.CreateClient(1)
	defer r.Close()

	val, err := r.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = r.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), duration).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":                  "Rate limit exceeded",
				"X-Rate-Limit-Remaining": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	//Check if the input is an actual url
	if !govalidator.IsURL(req.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}

	//Check for domain loop error
	if !helpers.RemoveDomainError(req.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Domain is not allowed"})
	}

	//Enforce http:// if not provided
	req.URL = helpers.EnforceHTTP(req.URL)

	//If custom short is not provided, generate a random one
	var id string
	if req.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = req.CustomShort
	}

	//Create database client for link storage
	r2 := database.CreateClient(2)
	defer r2.Close()

	val = r2.HGet(database.Ctx, "private", id).Val()
	val += r2.HGet(database.Ctx, "public", id).Val()
	if val != "" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Custom short already exists"})
	}

	//If expiry is not provided, set it to 24 hours
	if req.Expiry == 0 {
		req.Expiry = 24
	}
	//Store the link in the database
	//If the link is public, store it in the public hash
	if req.Public {
		//If the client is authorized, store the link in the public hash
		if helpers.AuthorizePublicUser(req.Username, req.SecretKey) {
			err = r2.HSet(database.Ctx, "public", id, req.URL).Err()
			r2.HExpire(database.Ctx, "public", req.Expiry*time.Hour, id)
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		// Else store the link in the private hash
	} else {
		err = r2.HSet(database.Ctx, "private", id, req.URL).Err()
		r2.HExpire(database.Ctx, "private", req.Expiry*time.Hour, id)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to connect to server"})
	}
	//Create response object
	resp := UrlRes{
		URL:             req.URL,
		CustomShort:     "",
		Expiry:          req.Expiry,
		XRateRemaining:  rateQuota,
		XRateLimitReset: duration,
	}
	//Decrement the rate limit
	r.Decr(database.Ctx, c.IP())
	//Set the response headers
	val, _ = r.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id
	//Return the response
	return c.Status(fiber.StatusOK).JSON(resp)
}
