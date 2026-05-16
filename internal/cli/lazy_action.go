package cli

import "fmt"

type lazyActionLoader func() (map[string]any, error)

func createLazyAction(loader lazyActionLoader, exportName string) func(args ...string) error {
	return func(args ...string) error {
		exports, err := loader()
		if err != nil {
			return err
		}
		exported, ok := exports[exportName]
		if !ok {
			return fmt.Errorf("lazy action export %q not found", exportName)
		}
		action, ok := exported.(func(...string) error)
		if !ok {
			return fmt.Errorf("lazy action export %q is not a function", exportName)
		}
		return action(args...)
	}
}
