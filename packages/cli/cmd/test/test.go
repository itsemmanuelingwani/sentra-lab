package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sentra-lab/cli/internal/config"
	"github.com/sentra-lab/cli/internal/grpc"
	"github.com/sentra-lab/cli/internal/reporter"
	"github.com/sentra-lab/cli/internal/ui"
	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

type TestCommand struct {
	logger       *utils.Logger
	configLoader *config.Loader
	engineClient *grpc.EngineClient
	reporter     reporter.Reporter
	parallel
