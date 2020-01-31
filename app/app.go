package app

import (
	"os"

	"github.com/alexej-v/grpc_cli/cli"
	"github.com/alexej-v/grpc_cli/config"
	"github.com/alexej-v/grpc_cli/proto"
)

type app struct {
	cfg  *config.Config
	spec proto.Spec
}

func (a *app) initConfig() (err error) {
	a.cfg, err = config.Init(os.Args)
	return err
}

func (a *app) initSpec() (err error) {
	a.spec, err = proto.Parse(a.cfg.Default.ProtoFile, a.cfg.Default.ProtoPath)
	return err
}

func Run() (err error) {
	newApp := new(app)

	if err = newApp.initConfig(); err != nil {
		return err
	}

	if err = newApp.initSpec(); err != nil {
		return err
	}

	if newApp.cfg.Describe != "" {
		newApp.spec.Describe(newApp.cfg)
		return nil
	}

	return cli.Run(cli.DefaultConfig(newApp.cfg, newApp.spec))
}
