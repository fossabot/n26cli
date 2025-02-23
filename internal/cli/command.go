package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nhatthm/n26cli/internal/app"
	"github.com/nhatthm/n26cli/internal/fmt"
	"github.com/nhatthm/n26cli/internal/service"
	"github.com/nhatthm/n26cli/internal/service/configurator"
)

// newAPICommand creates a new API command and decorates it with some global flags.
func newAPICommand(l *service.Locator, newCommand func(l *service.Locator) *cobra.Command) *cobra.Command {
	var apiCfg apiConfig

	cmd := newCommand(l)

	cmd.SetIn(l.InOrStdin())
	cmd.SetOut(l.OutOrStdout())
	cmd.SetErr(l.ErrOrStderr())

	cmd.Flags().StringVarP(&apiCfg.Username, "username", "u", "", "n26 username")
	cmd.Flags().StringVarP(&apiCfg.Password, "password", "p", "", "n26 password")
	cmd.Flags().StringVar(&apiCfg.Format, "format", "", "output format")

	run := runner(cmd)

	// If there is no runner, we do not have to setup the service locator.
	if run == nil {
		return cmd
	}

	cmd.RunE = nil
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := makeLocator(l, apiCfg); err != nil {
			handleErr(cmd, err)

			return
		}

		run(cmd, args)
	}

	return cmd
}

func runner(cmd *cobra.Command) func(cmd *cobra.Command, args []string) {
	if cmd.RunE == nil {
		return cmd.Run
	}

	fn := cmd.RunE

	return func(cmd *cobra.Command, args []string) {
		if err := fn(cmd, args); err != nil {
			handleErr(cmd, err)
		}
	}
}

func makeLocator(l *service.Locator, apiCfg apiConfig) error {
	c, err := configurator.New(rootCfg.ConfigFile).SafeRead()
	if err != nil {
		return err
	}

	if apiCfg.Format != "" {
		c.OutputFormat = apiCfg.Format
	} else if c.OutputFormat == "" {
		c.OutputFormat = service.OutputFormatPrettyJSON
	}

	if l.Config.Log.Level != 0 {
		c.Log.Level = l.Config.Log.Level
	} else {
		c.Log.Level = logLevel()
	}

	c.Log.Output = l.ErrOrStderr()
	c.N26.BaseURL = l.Config.N26.BaseURL
	c.N26.Username = apiCfg.Username
	c.N26.Password = apiCfg.Password
	c.N26.MFAWait = l.Config.N26.MFAWait
	c.N26.MFATimeout = l.Config.N26.MFATimeout

	l.Config = c

	return app.MakeServiceLocator(l)
}

func logLevel() zapcore.Level {
	if rootCfg.Debug {
		return zap.DebugLevel
	}

	if rootCfg.Verbose {
		return zap.InfoLevel
	}

	return zap.WarnLevel
}

func handleErr(fmt fmt.Fmt, err error) {
	if rootCfg.Debug {
		panic(err)
	}

	fmt.PrintErrln(err)
}
