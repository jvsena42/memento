package bot

import (
	"github.com/jvsena42/memento/internal/config"
	"github.com/jvsena42/memento/internal/storage"
	"github.com/jvsena42/memento/internal/twitter"
)

type Handler struct {
	Client       *twitter.Client
	CapsuleStore *storage.CapsuleStore
	Config       *config.Config
}
