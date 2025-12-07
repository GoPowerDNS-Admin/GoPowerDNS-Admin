package stdlogger

import (
	"github.com/rs/zerolog/log"
)

// Logger implements a standard compatible logger interface mapping logging func to zerolog.
type Logger interface {
	Errorf(string, ...interface{})
	Warningf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
}

// Adapter implements a zerolog compatibility logger.
type Adapter struct {
	Logger
}

// Errorf returns error.
func (a *Adapter) Errorf(f string, v ...interface{}) {
	log.Error().Msgf(f, v...)
}

// Warningf returns warning.
func (a *Adapter) Warningf(f string, v ...interface{}) {
	log.Warn().Msgf(f, v...)
}

// Infof returns info.
func (a *Adapter) Infof(f string, v ...interface{}) {
	log.Info().Msgf(f, v...)
}

// Debugf returns debug.
func (a *Adapter) Debugf(f string, v ...interface{}) {
	log.Debug().Msgf(f, v...)
}

// New returns a new logger adapter needed for packages using a std logger.
func New() Adapter {
	return Adapter{}
}
