package decorator

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	"app.ddd/internal/common/trace"
)

func generateActionName(value any) string {
	return strings.Split(reflect.TypeOf(value).String(), ".")[1]
}

func emitAction(ctx context.Context, kind, action string, value any, completed bool) {
	message := "Executing " + kind
	if completed {
		message = strings.ToUpper(kind[:1]) + kind[1:] + " executed successfully"
	}
	log.Debug().
		Str("traceId", trace.GetTraceId(ctx)).
		Str(kind, action).
		Str(kind+"_body", fmt.Sprintf("%#v", value)).
		Msg(message)
}
