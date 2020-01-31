package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/alexej-v/grpc_cli/client"
	"github.com/alexej-v/grpc_cli/config"
	"github.com/alexej-v/grpc_cli/grpc"
	"github.com/alexej-v/grpc_cli/proto"

	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

const (
	defaultPrompt          = "\033[32m> \033[39m"
	defaultInterruptPrompt = "^C"
	defaultEOFPrompt       = "exit"
	lineDelimiter          = " "
)

var defaultCompleter = readline.NewPrefixCompleter(
	readline.PcItem("info"),
	readline.PcItem("set",
		readline.PcItem("host"),
		readline.PcItem("port"),
		readline.PcItem("header"),
	),
)

// cliConfig short version of readline config
type cliConfig struct {
	Completer       *readline.PrefixCompleter
	Prompt          string
	InterruptPrompt string
	EOFPrompt       string

	appCfg  *config.Config
	spec    proto.Spec
	rlI     *readline.Instance
	headers map[string]string
}

// DefaultConfig returns default config
func DefaultConfig(appCfg *config.Config, spec proto.Spec) (cli *cliConfig) {
	cli = &cliConfig{
		Completer:       defaultCompleter,
		Prompt:          defaultPrompt,
		InterruptPrompt: defaultInterruptPrompt,
		EOFPrompt:       defaultEOFPrompt,

		appCfg: appCfg,
		spec:   spec,
	}
	cli.updateCompleterFromSpec(spec)
	return
}

// Run runs cli
func Run(cfg *cliConfig) error {
	rlI, err := readline.NewEx(&readline.Config{
		AutoComplete:    cfg.Completer,
		Prompt:          cfg.Prompt,
		InterruptPrompt: cfg.InterruptPrompt,
		EOFPrompt:       cfg.EOFPrompt,
	})
	if err != nil {
		return err
	}
	defer rlI.Close()
	cfg.rlI = rlI

	for {
		l, err := rlI.Readline()
		if err == io.EOF {
			return nil
		}
		if err == readline.ErrInterrupt && len(l) == 0 {
			return nil
		}
		cmdSlice := strings.Split(strings.TrimSpace(l), lineDelimiter)
		if cmdSlice == nil {
			continue
		}

		switch cmdSlice[0] {
		case "info":
			cfg.showInfo()
		case "package":
			cfg.getOrSetPackage(cmdSlice[1:])
		case "service":
			cfg.getOrSetService(cmdSlice[1:])
		case "call":
			cfg.call(cmdSlice[1:])
		case "set":
			cfg.setServerProps(cmdSlice[1:])
		default:
			// do nothing
		}
	}
}

func (c *cliConfig) showInfo() {
	c.Infof(
		"Host: %+v\nPort: %+v\nHeaders: %+v",
		c.appCfg.Server.Host, c.appCfg.Server.Port, c.headers,
	)
}

func (c *cliConfig) setServerProps(cmd []string) {
	if len(cmd) < 2 {
		return
	}
	switch cmd[0] {
	case "host":
		c.appCfg.Server.Host = cmd[1]
	case "port":
		c.appCfg.Server.Port = cmd[1]
	case "header":
		if len(cmd) > 2 {
			if c.headers == nil {
				c.headers = make(map[string]string)
			}
			c.headers[cmd[1]] = strings.Join(cmd[2:], lineDelimiter)
		}
	}
	c.showInfo()
}

func (c *cliConfig) getOrSetPackage(cmd []string) {
	if len(cmd) == 0 {
		c.Infof(c.appCfg.Default.Package)
		return
	}
	defer c.updPrompt()

	pkgExists := false
	for _, pkgName := range c.spec.PackageNames() {
		if pkgExists = pkgName == cmd[0]; pkgExists {
			break
		}
	}
	if !pkgExists {
		c.Errorf("unknown package name \"%s\"", cmd[0])
		return
	}
	c.appCfg.Default.Package = cmd[0]
	c.Infof(c.appCfg.Default.Package)
}

func (c *cliConfig) getOrSetService(cmd []string) {
	if len(cmd) == 0 {
		c.Infof(c.appCfg.Default.Service)
		return
	}
	defer c.updPrompt()

	svcNames, err := c.spec.ServiceNames(c.appCfg.Default.Package)
	if err != nil {
		c.Errorf(err.Error())
	}
	svcExists := false
	for _, svcName := range svcNames {
		if svcExists = svcName == cmd[0]; svcExists {
			break
		}
	}
	if !svcExists {
		c.Errorf("unknown service name \"%s\"", cmd[0])
	}
	c.appCfg.Default.Service = cmd[0]
	c.Infof(c.appCfg.Default.Service)
}

func (c *cliConfig) updateCompleterFromSpec(spec proto.Spec) {
	packageNames := make([]readline.PrefixCompleterInterface, 0)
	serviceNames := make([]readline.PrefixCompleterInterface, 0)
	methodNames := make([]readline.PrefixCompleterInterface, 0)

	for _, pkgName := range spec.PackageNames() {
		packageNames = append(packageNames, readline.PcItem(pkgName))
		svcNames, err := spec.ServiceNames(pkgName)
		if err != nil {
			continue
		}
		for _, svcName := range svcNames {
			serviceNames = append(serviceNames, readline.PcItem(svcName))
			gRPCs, err := spec.RPCs(pkgName, svcName)
			if err != nil {
				continue
			}
			for _, gRPC := range gRPCs {
				methodNames = append(methodNames, readline.PcItem(gRPC.Name))
			}

		}
	}

	c.Completer.SetChildren([]readline.PrefixCompleterInterface{
		readline.PcItem("package", packageNames...),
		readline.PcItem("service", serviceNames...),
		readline.PcItem("call", methodNames...),
		readline.PcItem("info"),
		readline.PcItem("set",
			readline.PcItem("host"),
			readline.PcItem("port"),
			readline.PcItem("header"),
		),
	})
}

func (c *cliConfig) updPrompt() {
	var prompt string
	if c.appCfg.Default.Package == "" || c.appCfg.Default.Package == "nil" {
		return
	}
	prompt = c.appCfg.Default.Package
	if c.appCfg.Default.Service == "" || c.appCfg.Default.Service == "nil" {
		c.rlI.SetPrompt(fmt.Sprintf("\033[34m%s \033[32m>\033[39m ", prompt))
		return
	}
	c.rlI.SetPrompt(fmt.Sprintf("\033[34m%s.%s \033[32m>\033[39m ", prompt, c.appCfg.Default.Service))
}

func (c *cliConfig) call(cmd []string) {
	if len(cmd) < 2 {
		return
	}
	cli, err := client.NewClient(&client.ClientCfg{
		Addr: c.appCfg.Server.Address(),
	})
	if err != nil {
		c.Errorf("failed to create new client: %v", err)
		return
	}

	rpc, err := c.spec.RPC(c.appCfg.Default.Package, c.appCfg.Default.Service, cmd[0])
	if err != nil {
		c.Errorf("failed to get RPC: %v", err)
		return
	}

	req, err := newGRPCRequest(rpc, strings.Join(cmd[1:], lineDelimiter), c.PrintJSON)
	if err != nil {
		c.Errorf(err.Error())
		return
	}

	resp, err := rpc.ResponseType.New()
	if err != nil {
		c.Errorf("failed to create new RPC response: %v", err)
		return
	}

	meta := metadata.New(nil)
	for headerName, headerBody := range c.headers {
		meta.Append(headerName, headerBody)
	}
	ctx := metadata.NewOutgoingContext(context.Background(), meta)
	// metadata.New(map[string]string{"authorization": "Bearer " + "eyJhbGciOiJFUzUxMiIsImtpZCI6IiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NzcxODg2NTQsImlhdCI6MTU3NzE4Njg1NCwiaXNzIjoib3pvbi50cmF2ZWwiLCJvenRpZCI6IjEzNzM1OTUwNjAwMCIsIm96dHNpZCI6ImY3ZDI5ZjY0LWM5ZmItNDEwNC05YjBiLTliOWY5YjYyMWQ3NSJ9.APDd7kRPa93jTRsMCnzOtqmXvPdOMOnu8l4vFFnM7dnqEBoT9xz4Z76fhARE-ZbwaSxo2eyrToiHp9J7ESOZ3HfMAZR64LIEMLE6jbhiU-q3XAB8a4g-hXYC3kSE8lMHlZoZsMgsKkHLmg5aOXNpd2ruIqJZQNHUALUql0vY1sfCYCjs"}),
	if err = cli.Invoke(ctx, rpc.FullyQualifiedName, req, resp); err != nil {
		c.Errorf("failed to request RPC service: %v", err)
		return
	}

	if err = c.PrintJSON(resp); err != nil {
		c.Errorf("failed to marshal RPC response: %v", err)
	}

}

func (c *cliConfig) Infof(format string, a ...interface{}) {
	fmt.Fprintf(c.rlI.Stdout(), fmt.Sprintf("\033[32m%s\033[39m\n", fmt.Sprintf(format, a...)))
}

func (c *cliConfig) Errorf(format string, a ...interface{}) {
	fmt.Fprintf(c.rlI.Stdout(), fmt.Sprintf("\033[31m%s\033[39m\n", fmt.Sprintf(format, a...)))
}

func (c *cliConfig) PrintJSON(j interface{}) error {
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(c.rlI.Stdout(), "%s\n", b)
	return nil
}

func newGRPCRequest(rpc *grpc.RPC, data string, printJSON func(interface{}) error) (interface{}, error) {
	req, err := rpc.RequestType.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new RPC request")
	}
	if err = json.Unmarshal([]byte(data), req); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal data \"%s\" to RPC request", data)
	}
	if err = printJSON(req); err != nil {
		return nil, errors.Wrapf(err, "failed to marshal RPC response")
	}
	return req, nil
}
