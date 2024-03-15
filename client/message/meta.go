/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package message

import (
	`context`

	`github.com/jhuix-go/ebus/discovery/common/metadata`
)

type mdMetaKey struct{}

func SizeMetadata(meta metadata.MD) (n int) {
	for k, vs := range meta {
		n += len(k) + 4
		for _, v := range vs {
			n += len(v) + 4
		}
	}
	return
}

func EncodeMetaContext(ctx context.Context, meta metadata.MD) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, mdMetaKey{}, meta)
}

func DecodeMetaContext(ctx context.Context) metadata.MD {
	if ctx == nil {
		return nil
	}

	if meta, ok := ctx.Value(mdMetaKey{}).(metadata.MD); ok {
		return meta
	}

	return nil
}

type mdContentKey struct{}

func EncodeContentContext(ctx context.Context, content interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, mdContentKey{}, content)
}

func DecodeContentContext(ctx context.Context) interface{} {
	if ctx == nil {
		return nil
	}

	return ctx.Value(mdContentKey{})
}

type ReqMetaKey struct{}

func EncodeReqMetaContext(ctx context.Context, meta map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ReqMetaKey{}, meta)
}

func DecodeReqMetaContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	if meta, ok := ctx.Value(ReqMetaKey{}).(map[string]string); ok {
		return meta
	}

	return nil
}

type ResMetaKey struct{}

func EncodeResMetaContext(ctx context.Context, meta map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ResMetaKey{}, meta)
}

func DecodeResMetaContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	if meta, ok := ctx.Value(ResMetaKey{}).(map[string]string); ok {
		return meta
	}

	return nil
}
