package eino_4

import (
	"context"
	"reflect"
	"testing"
)

func TestOrcGraphWithState(t *testing.T) {
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
			OrcGraphWithState(tt.args.ctx, tt.args.input)
		})
	}
}

func Test_genFunc(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *State
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genFunc(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}
