package awsmock

import (
	"context"
	"reflect"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
)

type AwsMockHandler struct {
	handlers []reflect.Value
	functors []reflect.Value
}

// NewAwsMockHandler - Create an AWS mocker to use with the AWS services, it returns an instrumented
// aws.Config that can be used to create AWS services.
// You can add as many individual request handlers as you need, as long as handlers
// correspond to the func(context.Context, <arg>)(<res>, error) format.
// E.g.:
// func(context.Context, *ec2.TerminateInstancesInput)(*ec2.TerminateInstancesOutput, error)
//
// You can also use a struct as the handler, in this case the AwsMockHandler will try
// to search for a method with a conforming signature.
func NewAwsMockHandler() *AwsMockHandler {
	return &AwsMockHandler{}
}

type retargetingHandler struct {
	parent *AwsMockHandler
}

func (f *retargetingHandler) ID() string {
	return "ShortCircuitRequest"
}

type initialRequestKey struct{}

func (f *retargetingHandler) HandleDeserialize(ctx context.Context, in middleware.DeserializeInput,
	next middleware.DeserializeHandler) (out middleware.DeserializeOutput, metadata middleware.Metadata, err error) {

	req := ctx.Value(&initialRequestKey{})
	out.Result, err = f.parent.invokeMethod(ctx, req)
	return
}

type saveRequestMiddleware struct {
}

func (s saveRequestMiddleware) ID() string {
	return "OriginalRequestSaver"
}

func (s saveRequestMiddleware) HandleInitialize(ctx context.Context, in middleware.InitializeInput,
	next middleware.InitializeHandler) (out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

	return next.HandleInitialize(context.WithValue(ctx, &initialRequestKey{}, in.Parameters), in)
}

func (a *AwsMockHandler) AwsConfig() aws.Config {
	cfg := aws.NewConfig()
	cfg.Region = "us-mars-1"
	cfg.APIOptions = []func(*middleware.Stack) error{
		func(stack *middleware.Stack) error {
			// We leave the serialization middleware intact in the vain hope that
			// AWS re-adds validation to serialization.
			//stack.Initialize.Clear()
			//stack.Serialize.Clear()

			// Make sure to save the initial non-serialized request
			_ = stack.Initialize.Add(&saveRequestMiddleware{}, middleware.Before)

			// Clear all the other middleware
			stack.Build.Clear()
			stack.Finalize.Clear()
			stack.Deserialize.Clear()

			// And replace the last one with our special middleware that dispatches
			// the request to our handlers
			_ = stack.Deserialize.Add(&retargetingHandler{parent: a}, middleware.Before)
			return nil
		},
	}

	return *cfg
}


func (a *AwsMockHandler) AddHandler(handlerObject interface {}) {
	handler := reflect.ValueOf(handlerObject)
	tp := handler.Type()

	if handler.Kind() == reflect.Func {
		PanicIfF(tp.NumOut() != 2 || tp.NumIn() != 2,
			"handler must have signature of func(context.Context, <arg>)(<res>, error)")
		a.functors = append(a.functors, handler)
	} else {
		PanicIfF(tp.NumMethod() == 0, "the handler must have invokable methods")
		a.handlers = append(a.handlers, handler)
	}
}

func (a *AwsMockHandler) invokeMethod(ctx context.Context,
	params interface{}) (interface{}, error) {

	for _, h := range a.handlers {
		for i := 0; i < h.NumMethod(); i++ {
			method := h.Method(i)

			matched, res, err := tryInvoke(ctx, params, method)
			if matched {
				return res, err
			}
		}
	}

	for _, f := range a.functors {
		matched, res, err := tryInvoke(ctx, params, f)
		if matched {
			return res, err
		}
	}

	panic("could not find a handler for operation: " + awsmiddleware.GetOperationName(ctx))
}

func tryInvoke(ctx context.Context, params interface{}, method reflect.Value) (
	bool, interface{}, error) {

	paramType := reflect.TypeOf(params)
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	methodDesc := method.Type()
	if methodDesc.NumIn() != 2 || methodDesc.NumOut() != 2 {
		return false, nil, nil
	}

	if !contextType.ConvertibleTo(methodDesc.In(0)) {
		return false, nil, nil
	}
	if !paramType.ConvertibleTo(methodDesc.In(1)) {
		return false, nil, nil
	}
	if !methodDesc.Out(1).ConvertibleTo(errorType) {
		return false, nil, nil
	}

	// It's our target!
	res := method.Call([]reflect.Value{reflect.ValueOf(ctx),
		reflect.ValueOf(params)})

	if !res[1].IsNil() {
		return true, nil, res[1].Interface().(error)
	}

	return true, res[0].Interface(), nil
}
