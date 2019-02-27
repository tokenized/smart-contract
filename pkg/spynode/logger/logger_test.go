package logger

import (
	"context"
	"os"
	"testing"
)

func TestLogger(test *testing.T) {
	showsystem := "showsystem"
	hidesystem := "hidesystem"
	fileName := "test.log"

	{
		os.Remove(fileName)
		logConfig := NewDevelopmentConfig()
		logConfig.EnableSubSystem(showsystem)
		logConfig.Main.AddFile(fileName)
		// logConfig.Main.Format |= System
		ctx := ContextWithLogConfig(context.Background(), logConfig)

		Log(ctx, Info, "First main entry")

		showCtx := ContextWithLogSubSystem(ctx, showsystem)
		Log(showCtx, Info, "First Sub entry")

		hideCtx := ContextWithLogSubSystem(ctx, hidesystem)
		Log(hideCtx, Info, "First Hidden Sub entry. You should not see this!")

		Log(ctx, Info, "Second main entry")
	}
}
