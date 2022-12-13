package finalizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsString(t *testing.T) {
	t.Parallel()

	type args struct {
		slice []string
		s     string
	}

	tests := []struct {
		name string
		args args
		want assert.BoolAssertionFunc
	}{
		{
			name: "should contain string",
			args: args{
				slice: []string{"s1", "s2", "s3"},
				s:     "s2",
			},
			want: assert.True,
		},
		{
			name: "should not contain string",
			args: args{
				slice: []string{"s1", "s2", "s3"},
				s:     "s4",
			},
			want: assert.False,
		},
		{
			name: "should process nil slice",
			args: args{
				slice: nil,
				s:     "",
			},
			want: assert.False,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.want(t, ContainsString(tt.args.slice, tt.args.s))
		})
	}
}

func TestRemoveString(t *testing.T) {
	t.Parallel()

	type args struct {
		slice []string
		s     string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "should remove string",
			args: args{
				slice: []string{"s1", "s2", "s3"},
				s:     "s2",
			},
			want: []string{"s1", "s3"},
		},
		{
			name: "should remove nothing",
			args: args{
				slice: []string{"s1", "s2", "s3"},
				s:     "s4",
			},
			want: []string{"s1", "s2", "s3"},
		},
		{
			name: "should work with nil slice",
			args: args{
				slice: nil,
				s:     "",
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, RemoveString(tt.args.slice, tt.args.s))
		})
	}
}
