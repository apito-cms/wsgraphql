package wsgraphql

import (
	"context"
	"fmt"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/tailor-inc/graphql"
	"github.com/tailor-inc/graphql/gqlerrors"
	"github.com/tailor-inc/graphql/language/ast"
	"github.com/tailor-inc/graphql/language/parser"
	"github.com/tailor-inc/graphql/language/source"
)

func (server *serverImpl) handleExtensionsInits(p *graphql.Params) *graphql.Result {
	var errs gqlerrors.FormattedErrors

	for _, ext := range server.extensions {
		func() {
			defer func() {
				if r := recover(); r != nil {
					errs = append(errs, gqlerrors.FormatError(fmt.Errorf("%s.Init: %v", ext.Name(), r)))
				}
			}()

			p.Context = ext.Init(p.Context, p)
		}()
	}

	if len(errs) == 0 {
		return nil
	}

	return &graphql.Result{
		Errors: errs,
	}
}

func (server *serverImpl) handleExtensionsParseDidStart(
	p *graphql.Params,
) (res *graphql.Result, endfn func(err error) *graphql.Result) {
	fs := make(map[string]graphql.ParseFinishFunc)

	var errs gqlerrors.FormattedErrors

	for _, ext := range server.extensions {
		var (
			ctx      context.Context
			finishFn graphql.ParseFinishFunc
		)

		func() {
			defer func() {
				if r := recover(); r != nil {
					errs = append(
						errs,
						gqlerrors.FormatError(fmt.Errorf("%s.ParseDidStart: %v", ext.Name(), r)),
					)
				}
			}()

			ctx, finishFn = ext.ParseDidStart(p.Context)

			p.Context = ctx

			fs[ext.Name()] = finishFn
		}()
	}

	endfn = func(err error) *graphql.Result {
		var inerrs gqlerrors.FormattedErrors

		if err != nil {
			inerrs = append(inerrs, gqlerrors.FormatError(err))
		}

		for name, fn := range fs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						inerrs = append(
							inerrs,
							gqlerrors.FormatError(fmt.Errorf("%s.ParseFinishFunc: %v", name, r)),
						)
					}
				}()

				fn(err)
			}()
		}

		if len(inerrs) == 0 {
			return nil
		}

		return &graphql.Result{
			Errors: inerrs,
		}
	}

	if len(errs) > 0 {
		res = &graphql.Result{
			Errors: errs,
		}
	}

	return
}

func (server *serverImpl) handleExtensionsValidationDidStart(
	p *graphql.Params,
) (errs []gqlerrors.FormattedError, endfn func(errs []gqlerrors.FormattedError) []gqlerrors.FormattedError) {
	fs := make(map[string]graphql.ValidationFinishFunc)

	for _, ext := range server.extensions {
		var (
			ctx      context.Context
			finishFn graphql.ValidationFinishFunc
		)

		func() {
			defer func() {
				if r := recover(); r != nil {
					errs = append(
						errs,
						gqlerrors.FormatError(fmt.Errorf("%s.ValidationDidStart: %v", ext.Name(), r)),
					)
				}
			}()

			ctx, finishFn = ext.ValidationDidStart(p.Context)

			p.Context = ctx
			fs[ext.Name()] = finishFn
		}()
	}

	endfn = func(errs []gqlerrors.FormattedError) (inerrs []gqlerrors.FormattedError) {
		inerrs = append(inerrs, errs...)

		for name, finishFn := range fs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						inerrs = append(
							inerrs,
							gqlerrors.FormatError(fmt.Errorf("%s.ValidationFinishFunc: %v", name, r)),
						)
					}
				}()

				finishFn(errs)
			}()
		}

		return
	}

	return
}

func (server *serverImpl) parseAST(
	ctx context.Context,
	payload *apollows.PayloadOperation,
) (err error) {
	opctx := OperationContext(ctx)

	var result *graphql.Result

	defer func() {
		if result != nil {
			err = ResultError{Result: result}
		}
	}()

	src := source.NewSource(&source.Source{
		Body: []byte(payload.Query),
		Name: "GraphQL request",
	})

	params := graphql.Params{
		Schema:         server.schema,
		RequestString:  payload.Query,
		RootObject:     server.rootObject,
		VariableValues: payload.Variables,
		OperationName:  payload.OperationName,
		Context:        ctx,
	}

	opctx.Set(ContextKeyOperationParams, &params)

	result = server.handleExtensionsInits(&params)
	if result != nil {
		return
	}

	var parseFinishFn func(err error) *graphql.Result

	result, parseFinishFn = server.handleExtensionsParseDidStart(&params)
	if result != nil {
		return
	}

	astdoc, err := parser.Parse(parser.ParseParams{Source: src})

	opctx.Set(ContextKeyAST, astdoc)

	result = parseFinishFn(err)
	if result != nil {
		return
	}

	errs, validationFinishFn := server.handleExtensionsValidationDidStart(&params)

	validationResult := graphql.ValidateDocument(&params.Schema, astdoc, nil)

	errs = append(errs, validationFinishFn(validationResult.Errors)...)

	if len(errs) > 0 || !validationResult.IsValid {
		result = &graphql.Result{
			Errors: errs,
		}

		return
	}

	var subscription bool

	for _, definition := range astdoc.Definitions {
		op, ok := definition.(*ast.OperationDefinition)
		if !ok {
			continue
		}

		if op.Operation == ast.OperationTypeSubscription {
			subscription = true

			break
		}
	}

	opctx.Set(ContextKeySubscription, subscription)

	return
}
