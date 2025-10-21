package eino_4

import "testing"

func TestOrcGraph(t *testing.T) {
	type args struct {
		choice string
	}
	tests := []struct {
		name string
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			OrcGraph(tt.args.choice)
		})
	}
}
