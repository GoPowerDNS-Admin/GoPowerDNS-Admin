package group

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
)

// saveStepError is a failed step inside a group save transaction. message is
// shown to the user; err is the cause and is only logged.
type saveStepError struct {
	message string
	err     error
}

func (e *saveStepError) Error() string { return e.message + ": " + e.err.Error() }

func (e *saveStepError) Unwrap() error { return e.err }

// abortGroupSave rolls back the transaction once and renders the error page
// for the failed step.
func abortGroupSave(c fiber.Ctx, tx *gorm.DB, err error) error {
	tx.Rollback()
	log.Error().Err(err).Msg("failed to update group")

	message := "Failed to update group"

	var failure *saveStepError
	if errors.As(err, &failure) {
		message = failure.message
	}

	return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", message, nil)
}
