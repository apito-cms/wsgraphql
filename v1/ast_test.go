package wsgraphql

import (
	"context"
	"testing"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/stretchr/testify/assert"
	"github.com/tailor-inc/graphql"
	"github.com/tailor-inc/graphql/gqlerrors"
)

func TestASTParse(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err = impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `query { foo }`,
		Variables:     nil,
		OperationName: "",
	})

	assert.Nil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseSubscription(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name:       "SubscriptionRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err = impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `subscription { foo }`,
		Variables:     nil,
		OperationName: "",
	})

	assert.Nil(t, err)
	assert.True(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

type testExt struct {
	initFn                 func(ctx context.Context, p *graphql.Params) context.Context
	hasResultFn            func() bool
	getResultFn            func(context.Context) interface{}
	parseDidStartFn        func(ctx context.Context) (context.Context, graphql.ParseFinishFunc)
	validationDidStartFn   func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc)
	executionDidStartFn    func(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc)
	resolveFieldDidStartFn func(
		ctx context.Context,
		i *graphql.ResolveInfo,
	) (context.Context, graphql.ResolveFieldFinishFunc)
	name string
}

func (t *testExt) Init(ctx context.Context, p *graphql.Params) context.Context {
	return t.initFn(ctx, p)
}

func (t *testExt) Name() string {
	return t.name
}

func (t *testExt) HasResult() bool {
	return t.hasResultFn()
}

func (t *testExt) GetResult(ctx context.Context) interface{} {
	return t.getResultFn(ctx)
}

func (t *testExt) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	return t.parseDidStartFn(ctx)
}

func (t *testExt) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	return t.validationDidStartFn(ctx)
}

func (t *testExt) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	return t.executionDidStartFn(ctx)
}

func (t *testExt) ResolveFieldDidStart(
	ctx context.Context,
	i *graphql.ResolveInfo,
) (context.Context, graphql.ResolveFieldFinishFunc) {
	return t.resolveFieldDidStartFn(ctx, i)
}

func testAstParseExtensions(
	t *testing.T,
	opctx mutable.Context,
	f func(ext *testExt),
) (err error) {
	text := &testExt{
		name: "foo",
		initFn: func(ctx context.Context, p *graphql.Params) context.Context {
			return ctx
		},
		hasResultFn: func() bool {
			return true
		},
		getResultFn: func(ctx context.Context) interface{} {
			return nil
		},
		parseDidStartFn: func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			return ctx, func(err error) {

			}
		},
		validationDidStartFn: func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			return ctx, func(errors []gqlerrors.FormattedError) {

			}
		},
		executionDidStartFn: func(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
			return ctx, func(result *graphql.Result) {

			}
		},
		resolveFieldDidStartFn: func(
			ctx context.Context,
			i *graphql.ResolveInfo,
		) (context.Context, graphql.ResolveFieldFinishFunc) {
			return ctx, func(i interface{}, err error) {

			}
		},
	}

	f(text)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
		Extensions: []graphql.Extension{
			text,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	return impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `query { foo }`,
		Variables:     nil,
		OperationName: "",
	})
}

func TestASTParseExtensions(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {

	})

	assert.Nil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseExtensionsPanicInit(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {
		ext.initFn = func(ctx context.Context, p *graphql.Params) context.Context {
			panic(1)
		}
	})

	assert.NotNil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.Nil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseExtensionsPanicValidation(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {
		ext.validationDidStartFn = func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			panic(1)
		}
	})

	assert.NotNil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseExtensionsPanicValidationCb(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {
		ext.validationDidStartFn = func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			return ctx, func(errors []gqlerrors.FormattedError) {
				panic(1)
			}
		}
	})

	assert.NotNil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseExtensionsPanicParse(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {
		ext.parseDidStartFn = func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			panic(1)
		}
	})

	assert.NotNil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.Nil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}

func TestASTParseExtensionsPanicParseCb(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())
	opctx.Set(ContextKeyOperationContext, opctx)

	err := testAstParseExtensions(t, opctx, func(ext *testExt) {
		ext.parseDidStartFn = func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			return ctx, func(err error) {
				panic(1)
			}
		}
	})

	assert.NotNil(t, err)
	assert.False(t, ContextSubscription(opctx))
	assert.NotNil(t, ContextAST(opctx))
	assert.NotNil(t, ContextOperationParams(opctx))
}
