package flagparse

import (
	"flag"
	"fmt"
	"os"
)

// command struct stores a slice of pointers to all subcommands
// and the description of the main CLI.
type command struct {
	description string
	flags       *flag.FlagSet

	// Command subcommands
	subCommands []*subCommand
}

// A subcommand required a flags.FlagSet to be run when the action is invoked.
type subCommand struct {
	description string
	flags       *flag.FlagSet
	action      func(args []string) error
}

// Create a new command with description.
// Usage:
//
//	command := flagparse.NewCommand("A tool for generating sql schema")
func NewCommand(desc string) *command {
	return &command{
		description: desc,
		subCommands: []*subCommand{},
		flags:       flag.NewFlagSet("", flag.ExitOnError),
	}
}

func (c *command) Flags() *flag.FlagSet {
	return c.flags
}

/*
Creates a new subcommand and adds it to the command structure.

name: Name of the subcommand

desc: Desctiption of subcommand

action: Function called when the subcommand is called.

	command := flagparse.NewCommand("A tool for generating sql schema")

	genCmd := command.AddSubCommand("generate", "Generate schema", generate)
	connCmd := command.AddSubCommand("connect", "Connect to the database", connect)

	genCmd.StringVar(&model, "model", "", "Name of the model to run migrations against")
	connCmd.StringVar(&dsn, "dsn", "", "data source name")

	err := command.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
*/
func (c *command) AddSubCommand(name, description string, action func(args []string) error) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ExitOnError)
	c.subCommands = append(c.subCommands, &subCommand{
		description: description,
		action:      action,
		flags:       flagSet,
	})
	return flagSet
}

func helpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--h" || arg == "-help" || arg == "--help" {
			return true
		}
	}
	return false
}

// Parse must be called after creating all subcommands.
func (cs *command) Parse(args []string) error {
	// Parse main subcommands
	helpNeeded := helpRequested(args)
	if err := cs.flags.Parse(args); err != nil && !helpNeeded {
		return err
	}

	if len(args) < 1 || helpNeeded {
		cs.Usage()
		return nil
	}

	// Parse flags on subcommands
	for _, c := range cs.subCommands {
		if c.flags.Name() == args[0] {
			c.flags.Usage = func() { cs.usageFor(c) }
			c.flags.Parse(args[1:])
			return c.action(c.flags.Args())
		}
	}

	cs.Usage()
	return fmt.Errorf("unknown subcommand %q", args[0])
}

// Print the command Usage text.
func (cs *command) Usage() {
	fmt.Println(cs.description)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s command [arguments]\n", os.Args[0])
	fmt.Println()
	fmt.Println("Commands:")
	for _, c := range cs.subCommands {
		fmt.Printf("  %-15s %s\n", c.flags.Name(), c.description)
	}
	fmt.Println()
}

// Print subcommand usage text.
func (cs *command) usageFor(c *subCommand) {
	fmt.Printf("Usage: %s %s [flags]\n", os.Args[0], c.flags.Name())
	fmt.Println()
	fmt.Println("Flags:")
	c.flags.PrintDefaults()
	fmt.Println()
}
