package function_test

import (
	"context"
	function "handler/function"
	"log/slog"
	"os"
	"testing"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

func TestHandler(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	req := sdk.Request{
		"count": 2,
	}

	resp, err := function.Handle(context.TODO(), logger, req)
	if err != nil {
		t.Log("test error: ", err)
		t.Fatalf("error: %v", err)
	}
	t.Log("handler response: ", resp)

}
