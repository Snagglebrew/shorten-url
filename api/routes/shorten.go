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

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"X-Rate-Limit-Remaining"`
	XRateLimitReset time.Duration `json:"X-Rate-Limit-Reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	req := new(request)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	rateQuota, _ := strconv.Atoi(os.Getenv("API_QUOTA"))
	duration, _ := time.ParseDuration(os.Getenv("API_QUOTA_DURATION") + "h")

	//Rate Limiting
	r2 := database.CreateClient(1)
	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), duration).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
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
	r := database.CreateClient(0)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Custom short already exists"})
	}

	//If expiry is not provided, set it to 24 hours
	if req.Expiry == 0 {
		req.Expiry = 24
	}
	//Store the link in the database
	err = r.Set(database.Ctx, id, req.URL, req.Expiry*time.Hour).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to connect to server"})
	}
	//Create response object
	resp := response{
		URL:             req.URL,
		CustomShort:     "",
		Expiry:          req.Expiry,
		XRateRemaining:  rateQuota,
		XRateLimitReset: duration,
	}
	//Decrement the rate limit
	r2.Decr(database.Ctx, c.IP())
	//Set the response headers
	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id
	//Return the response
	return c.Status(fiber.StatusOK).JSON(resp)
}
