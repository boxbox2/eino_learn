package eino_4

import (
	"context"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/callbacks"
)

func TestOrcGraphWithCallback(t *testing.T) {
	type args struct {
		ctx   context.Context
		input map[string]string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			OrcGraphWithCallback(tt.args.ctx, tt.args.input)
		})
	}
}

func Test_genCallback(t *testing.T) {
	tests := []struct {
		name string
		want callbacks.Handler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genCallback(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genCallback() = %v, want %v", got, tt.want)
			}
		})
	}
}
