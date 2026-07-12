package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestNoShorthandCollisions guards against a subcommand declaring a flag whose
// single-letter shorthand collides with one of RootCmd's persistent shorthands
// (-p/--profile, -W/--workspace, -R/--repo). cobra panics on such a collision
// when it merges persistent flags into the subcommand at execution time, so a
// collision takes the command down on every invocation (see the -p/--pattern
// regression on `pipelines trigger`).
func TestNoShorthandCollisions(t *testing.T) {
	persistent := map[string]string{} // shorthand -> flag name
	RootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Shorthand != "" {
			persistent[f.Shorthand] = f.Name
		}
	})

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Shorthand == "" {
				return
			}
			if pName, ok := persistent[f.Shorthand]; ok {
				t.Errorf("%q flag --%s uses shorthand -%s, which collides with the persistent --%s (cobra will panic when flags merge)",
					c.CommandPath(), f.Name, f.Shorthand, pName)
			}
		})
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(RootCmd)
}
