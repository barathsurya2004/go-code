package activities

import (
	"context"
	"fmt"
)

func HelloWorldActivity(ctx context.Context, name string) (string, error) {
	fmt.Printf("Executing HelloWorldActivity for %s\n", name)

	return fmt.Sprintf("Hello %s\n", name), nil
}
