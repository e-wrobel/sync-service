package validators

import (
	"fmt"
	"os"
)

func MustDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("%s is not directory", path)
	}
	return nil
}
